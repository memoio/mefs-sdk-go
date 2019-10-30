/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2018 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package minio

import (
	"bytes"
	"crypto/md5"
	"hash"
	"io"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"runtime"
	"sync"

	"github.com/minio/sha256-simd"

	"github.com/memoio/minio-go/pkg/credentials"
	"github.com/memoio/minio-go/pkg/s3utils"

	"context"
	"errors"
	"fmt"
	"io/ioutil"
	gohttp "net/http"
	"os"
	"strings"
	"time"

	files "github.com/ipfs/go-ipfs-files"
	homedir "github.com/mitchellh/go-homedir"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

// Client implements Amazon S3 compatible methods.
type Client struct {
	///  Standard options.

	url string

	// Parsed endpoint url provided by the user.
	//endpointURL *url.URL

	// Holds various credential providers.
	credsProvider *credentials.Credentials

	// Custom signerType value overrides all credentials.
	overrideSignerType credentials.SignatureType

	// User supplied.
	appInfo struct {
		appName    string
		appVersion string
	}

	// Indicate whether we are using https or not
	secure bool

	// Needs allocation.
	httpClient     *http.Client
	bucketLocCache *bucketLocationCache

	// Advanced functionality.
	isTraceEnabled  bool
	traceErrorsOnly bool
	traceOutput     io.Writer

	// S3 specific accelerated endpoint.
	s3AccelerateEndpoint string

	// Region endpoint
	region string

	// Random seed.
	random *rand.Rand

	// lookup indicates type of url lookup supported by server. If not specified,
	// default to Auto.
	lookup BucketLookupType
}

// Options for New method
type Options struct {
	Creds        *credentials.Credentials
	Secure       bool
	Region       string
	BucketLookup BucketLookupType
	// Add future fields here
}

// Global constants.
const (
	libraryName    = "minio-go"
	libraryVersion = "v6.0.40"
)

// User Agent should always following the below style.
// Please open an issue to discuss any new changes here.
//
//       MinIO (OS; ARCH) LIB/VER APP/VER
const (
	libraryUserAgentPrefix = "MinIO (" + runtime.GOOS + "; " + runtime.GOARCH + ") "
	libraryUserAgent       = libraryUserAgentPrefix + libraryName + "/" + libraryVersion
)

// BucketLookupType is type of url lookup supported by server.
type BucketLookupType int

// Different types of url lookup supported by the server.Initialized to BucketLookupAuto
const (
	BucketLookupAuto BucketLookupType = iota
	BucketLookupDNS
	BucketLookupPath
)

// NewV2 - instantiate minio client with Amazon S3 signature version
// '2' compatibility.
func NewV2(endpoint string, accessKeyID, secretAccessKey string, secure bool) (*Client, error) {
	creds := credentials.NewStaticV2(accessKeyID, secretAccessKey, "")
	clnt, err := privateNew(endpoint, creds, secure, "", BucketLookupAuto)
	if err != nil {
		return nil, err
	}
	clnt.overrideSignerType = credentials.SignatureV2
	return clnt, nil
}

// NewV4 - instantiate minio client with Amazon S3 signature version
// '4' compatibility.
func NewV4(endpoint string, accessKeyID, secretAccessKey string, secure bool) (*Client, error) {
	creds := credentials.NewStaticV4(accessKeyID, secretAccessKey, "")
	clnt, err := privateNew(endpoint, creds, secure, "", BucketLookupAuto)
	if err != nil {
		return nil, err
	}
	clnt.overrideSignerType = credentials.SignatureV4
	return clnt, nil
}

// New - instantiate minio client, adds automatic verification of signature.
func New(endpoint, accessKeyID, secretAccessKey string, secure bool) (*Client, error) {
	creds := credentials.NewStaticV4(accessKeyID, secretAccessKey, "")
	clnt, err := privateNew(endpoint, creds, secure, "", BucketLookupAuto)
	if err != nil {
		return nil, err
	}
	return clnt, nil
}

// NewWithCredentials - instantiate minio client with credentials provider
// for retrieving credentials from various credentials provider such as
// IAM, File, Env etc.
func NewWithCredentials(endpoint string, creds *credentials.Credentials, secure bool, region string) (*Client, error) {
	return privateNew(endpoint, creds, secure, region, BucketLookupAuto)
}

// NewWithRegion - instantiate minio client, with region configured. Unlike New(),
// NewWithRegion avoids bucket-location lookup operations and it is slightly faster.
// Use this function when if your application deals with single region.
func NewWithRegion(endpoint, accessKeyID, secretAccessKey string, secure bool, region string) (*Client, error) {
	creds := credentials.NewStaticV4(accessKeyID, secretAccessKey, "")
	return privateNew(endpoint, creds, secure, region, BucketLookupAuto)
}

// NewWithOptions - instantiate minio client with options
func NewWithOptions(endpoint string, opts *Options) (*Client, error) {
	return privateNew(endpoint, opts.Creds, opts.Secure, opts.Region, opts.BucketLookup)
}

// lockedRandSource provides protected rand source, implements rand.Source interface.
type lockedRandSource struct {
	lk  sync.Mutex
	src rand.Source
}

// Int63 returns a non-negative pseudo-random 63-bit integer as an int64.
func (r *lockedRandSource) Int63() (n int64) {
	r.lk.Lock()
	n = r.src.Int63()
	r.lk.Unlock()
	return
}

// Seed uses the provided seed value to initialize the generator to a
// deterministic state.
func (r *lockedRandSource) Seed(seed int64) {
	r.lk.Lock()
	r.src.Seed(seed)
	r.lk.Unlock()
}

// Redirect requests by re signing the request.
func (c *Client) redirectHeaders(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return errors.New("stopped after 5 redirects")
	}
	if len(via) == 0 {
		return nil
	}
	lastRequest := via[len(via)-1]
	var reAuth bool
	for attr, val := range lastRequest.Header {
		// if hosts do not match do not copy Authorization header
		if attr == "Authorization" && req.Host != lastRequest.Host {
			reAuth = true
			continue
		}
		if _, ok := req.Header[attr]; !ok {
			req.Header[attr] = val
		}
	}

	//*c.endpointURL = *req.URL

	value, err := c.credsProvider.Get()
	if err != nil {
		return err
	}
	var (
		signerType = value.SignerType
	)

	// Custom signer set then override the behavior.
	if c.overrideSignerType != credentials.SignatureDefault {
		signerType = c.overrideSignerType
	}

	// If signerType returned by credentials helper is anonymous,
	// then do not sign regardless of signerType override.
	if value.SignerType == credentials.SignatureAnonymous {
		signerType = credentials.SignatureAnonymous
	}

	if reAuth {
		// Check if there is no region override, if not get it from the URL if possible.
		// if region == "" {
		// 	region = s3utils.GetRegionFromURL(*c.endpointURL)
		// }
		switch {
		case signerType.IsV2():
			return errors.New("signature V2 cannot support redirection")
			// case signerType.IsV4():
			// 	s3signer.SignV4(*req, accessKeyID, secretAccessKey, sessionToken, getDefaultLocation(*c.endpointURL, region))
		}
	}
	return nil
}

const (
	DefaultPathName = ".mefs"
	DefaultPathRoot = "~/" + DefaultPathName
	DefaultApiFile  = "api"
	EnvDir          = "MEFS_PATH"
)

func privateNew(endpoint string, creds *credentials.Credentials, secure bool, region string, lookup BucketLookupType) (*Client, error) {
	if region == "local" {
		baseDir := os.Getenv(EnvDir)
		if baseDir == "" {
			baseDir = DefaultPathRoot
		}

		baseDir, err := homedir.Expand(baseDir)
		if err != nil {
			return nil, err
		}

		apiFile := path.Join(baseDir, DefaultApiFile)

		if _, err := os.Stat(apiFile); err != nil {
			return nil, err
		}

		api, err := ioutil.ReadFile(apiFile)
		if err != nil {
			return nil, err
		}

		endpoint = strings.TrimSpace(string(api))
	}

	// instantiate new Client.
	clnt := new(Client)

	// Save the credentials.
	clnt.credsProvider = creds

	// Remember whether we are using https or not
	clnt.secure = secure

	// // Save endpoint URL, user agent for future uses.
	// clnt.endpointURL = endpointURL

	if a, err := ma.NewMultiaddr(endpoint); err == nil {
		_, host, err := manet.DialArgs(a)
		if err == nil {
			endpoint = host
		}
	}
	clnt.url = endpoint

	// Instantiate http client and bucket location cache.
	clnt.httpClient = &gohttp.Client{
		Transport: &gohttp.Transport{
			Proxy:             gohttp.ProxyFromEnvironment,
			DisableKeepAlives: true,
		},
	}
	// We don't support redirects.
	clnt.httpClient.CheckRedirect = func(_ *gohttp.Request, _ []*gohttp.Request) error {
		return fmt.Errorf("unexpected redirect")
	}

	return clnt, nil
}

// SetAppInfo - add application details to user agent.
func (c *Client) SetAppInfo(appName string, appVersion string) {
	// if app name and version not set, we do not set a new user agent.
	if appName != "" && appVersion != "" {
		c.appInfo.appName = appName
		c.appInfo.appVersion = appVersion
	}
}

// SetCustomTransport - set new custom transport.
func (c *Client) SetCustomTransport(customHTTPTransport http.RoundTripper) {
	// Set this to override default transport
	// ``http.DefaultTransport``.
	//
	// This transport is usually needed for debugging OR to add your
	// own custom TLS certificates on the client transport, for custom
	// CA's and certs which are not part of standard certificate
	// authority follow this example :-
	//
	//   tr := &http.Transport{
	//           TLSClientConfig:    &tls.Config{RootCAs: pool},
	//           DisableCompression: true,
	//   }
	//   api.SetCustomTransport(tr)
	//
	if c.httpClient != nil {
		c.httpClient.Transport = customHTTPTransport
	}
}

// TraceOn - enable HTTP tracing.
func (c *Client) TraceOn(outputStream io.Writer) {
	// if outputStream is nil then default to os.Stdout.
	if outputStream == nil {
		outputStream = os.Stdout
	}
	// Sets a new output stream.
	c.traceOutput = outputStream

	// Enable tracing.
	c.isTraceEnabled = true
}

// TraceErrorsOnlyOn - same as TraceOn, but only errors will be traced.
func (c *Client) TraceErrorsOnlyOn(outputStream io.Writer) {
	c.TraceOn(outputStream)
	c.traceErrorsOnly = true
}

// TraceErrorsOnlyOff - Turns off the errors only tracing and everything will be traced after this call.
// If all tracing needs to be turned off, call TraceOff().
func (c *Client) TraceErrorsOnlyOff() {
	c.traceErrorsOnly = false
}

// TraceOff - disable HTTP tracing.
func (c *Client) TraceOff() {
	// Disable tracing.
	c.isTraceEnabled = false
	c.traceErrorsOnly = false
}

// SetS3TransferAccelerate - turns s3 accelerated endpoint on or off for all your
// requests. This feature is only specific to S3 for all other endpoints this
// function does nothing. To read further details on s3 transfer acceleration
// please vist -
// http://docs.aws.amazon.com/AmazonS3/latest/dev/transfer-acceleration.html
func (c *Client) SetS3TransferAccelerate(accelerateEndpoint string) {
	// if s3utils.IsAmazonEndpoint(*c.endpointURL) {
	// 	c.s3AccelerateEndpoint = accelerateEndpoint
	// }
}

// Hash materials provides relevant initialized hash algo writers
// based on the expected signature type.
//
//  - For signature v4 request if the connection is insecure compute only sha256.
//  - For signature v4 request if the connection is secure compute only md5.
//  - For anonymous request compute md5.
func (c *Client) hashMaterials() (hashAlgos map[string]hash.Hash, hashSums map[string][]byte) {
	hashSums = make(map[string][]byte)
	hashAlgos = make(map[string]hash.Hash)
	if c.overrideSignerType.IsV4() {
		if c.secure {
			hashAlgos["md5"] = md5.New()
		} else {
			hashAlgos["sha256"] = sha256.New()
		}
	} else {
		if c.overrideSignerType.IsAnonymous() {
			hashAlgos["md5"] = md5.New()
		}
	}
	return hashAlgos, hashSums
}

// requestMetadata - is container for all the values to make a request.
type requestMetadata struct {
	// If set newRequest presigns the URL.
	presignURL bool

	// User supplied.
	bucketName   string
	objectName   string
	queryValues  url.Values
	customHeader http.Header
	expires      int64

	// Generated by our internal code.
	bucketLocation   string
	contentBody      io.Reader
	contentLength    int64
	contentMD5Base64 string // carries base64 encoded md5sum
	contentSHA256Hex string // carries hex encoded sha256sum
}

// dumpHTTP - dump HTTP request and response.
func (c Client) dumpHTTP(req *http.Request, resp *http.Response) error {
	// Starts http dump.
	_, err := fmt.Fprintln(c.traceOutput, "---------START-HTTP---------")
	if err != nil {
		return err
	}

	// Filter out Signature field from Authorization header.
	origAuth := req.Header.Get("Authorization")
	if origAuth != "" {
		req.Header.Set("Authorization", redactSignature(origAuth))
	}

	// Only display request header.
	reqTrace, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		return err
	}

	// Write request to trace output.
	_, err = fmt.Fprint(c.traceOutput, string(reqTrace))
	if err != nil {
		return err
	}

	// Only display response header.
	var respTrace []byte

	// For errors we make sure to dump response body as well.
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusPartialContent &&
		resp.StatusCode != http.StatusNoContent {
		respTrace, err = httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
	} else {
		respTrace, err = httputil.DumpResponse(resp, false)
		if err != nil {
			return err
		}
	}

	// Write response to trace output.
	_, err = fmt.Fprint(c.traceOutput, strings.TrimSuffix(string(respTrace), "\r\n"))
	if err != nil {
		return err
	}

	// Ends the http dump.
	_, err = fmt.Fprintln(c.traceOutput, "---------END-HTTP---------")
	if err != nil {
		return err
	}

	// Returns success.
	return nil
}

// do - execute http request.
func (c Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Handle this specifically for now until future Golang versions fix this issue properly.
		if urlErr, ok := err.(*url.Error); ok {
			if strings.Contains(urlErr.Err.Error(), "EOF") {
				return nil, &url.Error{
					Op:  urlErr.Op,
					URL: urlErr.URL,
					Err: errors.New("Connection closed by foreign host " + urlErr.URL + ". Retry again."),
				}
			}
		}
		return nil, err
	}

	// Response cannot be non-nil, report error if thats the case.
	if resp == nil {
		msg := "Response is empty. " + reportIssue
		return nil, ErrInvalidArgument(msg)
	}

	// If trace is enabled, dump http request and response,
	// except when the traceErrorsOnly enabled and the response's status code is ok
	if c.isTraceEnabled && !(c.traceErrorsOnly && resp.StatusCode == http.StatusOK) {
		err = c.dumpHTTP(req, resp)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// List of success status.
var successStatus = []int{
	http.StatusOK,
	http.StatusNoContent,
	http.StatusPartialContent,
}

// executeMethod - instantiates a given method, and retries the
// request upon any error up to maxRetries attempts in a binomially
// delayed manner using a standard back off algorithm.
func (c Client) executeMethod(ctx context.Context, method string, metadata requestMetadata) (res *http.Response, err error) {
	var isRetryable bool     // Indicates if request can be retried.
	var bodySeeker io.Seeker // Extracted seeker from io.Reader.
	var reqRetry = MaxRetry  // Indicates how many times we can retry the request

	if metadata.contentBody != nil {
		// Check if body is seekable then it is retryable.
		bodySeeker, isRetryable = metadata.contentBody.(io.Seeker)
		switch bodySeeker {
		case os.Stdin, os.Stdout, os.Stderr:
			isRetryable = false
		}
		// Retry only when reader is seekable
		if !isRetryable {
			reqRetry = 1
		}

		// Figure out if the body can be closed - if yes
		// we will definitely close it upon the function
		// return.
		bodyCloser, ok := metadata.contentBody.(io.Closer)
		if ok {
			defer bodyCloser.Close()
		}
	}

	// Create a done channel to control 'newRetryTimer' go routine.
	doneCh := make(chan struct{}, 1)

	// Indicate to our routine to exit cleanly upon return.
	defer close(doneCh)

	// Blank indentifier is kept here on purpose since 'range' without
	// blank identifiers is only supported since go1.4
	// https://golang.org/doc/go1.4#forrange.
	for range c.newRetryTimer(reqRetry, DefaultRetryUnit, DefaultRetryCap, MaxJitter, doneCh) {
		// Retry executes the following function body if request has an
		// error until maxRetries have been exhausted, retry attempts are
		// performed after waiting for a given period of time in a
		// binomial fashion.
		if isRetryable {
			// Seek back to beginning for each attempt.
			if _, err = bodySeeker.Seek(0, 0); err != nil {
				// If seek failed, no need to retry.
				return nil, err
			}
		}

		// Instantiate a new request.
		var req *http.Request

		// Add context to request
		req = req.WithContext(ctx)

		// Initiate the request.
		res, err = c.do(req)
		if err != nil {
			// For supported http requests errors verify.
			if isHTTPReqErrorRetryable(err) {
				continue // Retry.
			}
			// For other errors, return here no need to retry.
			return nil, err
		}

		// For any known successful http status, return quickly.
		for _, httpStatus := range successStatus {
			if httpStatus == res.StatusCode {
				return res, nil
			}
		}

		// Read the body to be saved later.
		errBodyBytes, err := ioutil.ReadAll(res.Body)
		// res.Body should be closed
		closeResponse(res)
		if err != nil {
			return nil, err
		}

		// Save the body.
		errBodySeeker := bytes.NewReader(errBodyBytes)
		res.Body = ioutil.NopCloser(errBodySeeker)

		// For errors verify if its retryable otherwise fail quickly.
		errResponse := ToErrorResponse(httpRespToErrorResponse(res, metadata.bucketName, metadata.objectName))

		// Save the body back again.
		errBodySeeker.Seek(0, 0) // Seek back to starting point.
		res.Body = ioutil.NopCloser(errBodySeeker)

		// Bucket region if set in error response and the error
		// code dictates invalid region, we can retry the request
		// with the new region.
		//
		// Additionally we should only retry if bucketLocation and custom
		// region is empty.
		if c.region == "" {
			switch errResponse.Code {
			case "AuthorizationHeaderMalformed":
				fallthrough
			case "InvalidRegion":
				fallthrough
			case "AccessDenied":
				if metadata.bucketName != "" && errResponse.Region != "" {
					// Gather Cached location only if bucketName is present.
					if _, cachedOk := c.bucketLocCache.Get(metadata.bucketName); cachedOk {
						c.bucketLocCache.Set(metadata.bucketName, errResponse.Region)
						continue // Retry.
					}
				} else {
					// Most probably for ListBuckets()
					if errResponse.Region != metadata.bucketLocation {
						// Retry if the error
						// response has a
						// different region
						// than the request we
						// just made.
						metadata.bucketLocation = errResponse.Region
						continue // Retry
					}
				}
			}
		}

		// Verify if error response code is retryable.
		if isS3CodeRetryable(errResponse.Code) {
			continue // Retry.
		}

		// Verify if http status code is retryable.
		if isHTTPStatusRetryable(res.StatusCode) {
			continue // Retry.
		}

		// For all other cases break out of the retry loop.
		break
	}
	return res, err
}

// set User agent.
func (c Client) setUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", libraryUserAgent)
	if c.appInfo.appName != "" && c.appInfo.appVersion != "" {
		req.Header.Set("User-Agent", libraryUserAgent+" "+c.appInfo.appName+"/"+c.appInfo.appVersion)
	}
}

// returns true if virtual hosted style requests are to be used.
func (c *Client) isVirtualHostStyleRequest(url url.URL, bucketName string) bool {
	if bucketName == "" {
		return false
	}

	if c.lookup == BucketLookupDNS {
		return true
	}
	if c.lookup == BucketLookupPath {
		return false
	}

	// default to virtual only for Amazon/Google  storage. In all other cases use
	// path style requests
	return s3utils.IsVirtualHostSupported(url, bucketName)
}

func (c *Client) SetTimeout(d time.Duration) {
	c.httpClient.Timeout = d
}

func (c *Client) Request(command string, args ...string) *RequestBuilder {
	return &RequestBuilder{
		command: command,
		args:    args,
		client:  c,
	}
}

type IdOutput struct {
	ID              string
	PublicKey       string
	Addresses       []string
	AgentVersion    string
	ProtocolVersion string
}

// ID gets information about a given peer.  Arguments:
//
// peer: peer.ID of the node to look up.  If no peer is specified,
//   return information about the local peer.
func (c *Client) ID(peer ...string) (*IdOutput, error) {
	if len(peer) > 1 {
		return nil, fmt.Errorf("Too many peer arguments")
	}

	var out IdOutput
	if err := c.Request("id", peer...).Exec(context.Background(), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

const (
	DirectPin    = "direct"
	RecursivePin = "recursive"
	IndirectPin  = "indirect"
)

type PeerInfo struct {
	Addrs []string
	ID    string
}

func (c *Client) FindPeer(peer string) (*PeerInfo, error) {
	var peers struct{ Responses []PeerInfo }
	err := c.Request("dht/findpeer", peer).Exec(context.Background(), &peers)
	if err != nil {
		return nil, err
	}
	if len(peers.Responses) == 0 {
		return nil, errors.New("peer not found")
	}
	return &peers.Responses[0], nil
}

func (c *Client) ResolvePath(path string) (string, error) {
	var out struct {
		Path string
	}

	err := c.Request("resolve", path).Exec(context.Background(), &out)
	if err != nil {
		return "", err
	}

	return strings.TrimPrefix(out.Path, "/ipfs/"), nil
}

// returns ipfs version and commit sha
func (c *Client) Version() (string, string, error) {
	ver := struct {
		Version string
		Commit  string
	}{}

	if err := c.Request("version").Exec(context.Background(), &ver); err != nil {
		return "", "", err
	}
	return ver.Version, ver.Commit, nil
}

func (c *Client) IsUp() bool {
	_, _, err := c.Version()
	return err == nil
}

func (c *Client) BlockStat(path string) (string, int, error) {
	var inf struct {
		Key  string
		Size int
	}

	if err := c.Request("block/stat", path).Exec(context.Background(), &inf); err != nil {
		return "", 0, err
	}
	return inf.Key, inf.Size, nil
}

func (c *Client) BlockGet(path string) ([]byte, error) {
	resp, err := c.Request("block/get", path).Send(context.Background())
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	if resp.Error != nil {
		return nil, resp.Error
	}

	return ioutil.ReadAll(resp.Output)
}

func (c *Client) BlockPut(block []byte, format, mhtype string, mhlen int) (string, error) {
	var out struct {
		Key string
	}

	fr := files.NewBytesFile(block)
	slf := files.NewSliceDirectory([]files.DirEntry{files.FileEntry("", fr)})
	fileReader := files.NewMultiFileReader(slf, true)

	return out.Key, c.Request("block/put").
		Option("mhtype", mhtype).
		Option("format", format).
		Option("mhlen", mhlen).
		Body(fileReader).
		Exec(context.Background(), &out)
}

type SwarmStreamInfo struct {
	Protocol string
}

type SwarmConnInfo struct {
	Addr    string
	Peer    string
	Latency string
	Muxer   string
	Streams []SwarmStreamInfo
}

type SwarmConnInfos struct {
	Peers []SwarmConnInfo
}

// SwarmPeers gets all the swarm peers
func (c *Client) SwarmPeers(ctx context.Context) (*SwarmConnInfos, error) {
	v := &SwarmConnInfos{}
	err := c.Request("swarm/peers").Exec(ctx, &v)
	return v, err
}

type swarmConnection struct {
	Strings []string
}

// SwarmConnect opens a swarm connection to a specific address.
func (c *Client) SwarmConnect(ctx context.Context, addr ...string) error {
	var conn *swarmConnection
	err := c.Request("swarm/connect").
		Arguments(addr...).
		Exec(ctx, &conn)
	return err
}
