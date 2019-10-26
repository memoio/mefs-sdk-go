package minio

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	peer "github.com/libp2p/go-libp2p-core/peer"
)

type UserPrivMessage struct {
	Address string
	Sk      string
}

type StringList struct {
	ChildLists []string
}

func (fl StringList) String() string {
	var buffer bytes.Buffer
	for i := 0; i < len(fl.ChildLists); i++ {
		buffer.WriteString(fl.ChildLists[i])
		buffer.WriteString("\n")
	}
	return buffer.String()
}

type IntList struct {
	ChildLists []int
}

func (fl IntList) String() string {
	var buffer bytes.Buffer
	for i := 0; i < len(fl.ChildLists); i++ {
		buffer.WriteString(strconv.Itoa(fl.ChildLists[i]))
		buffer.WriteString("\n")
	}
	return buffer.String()
}

type LfsOpts = func(*RequestBuilder) error

func SetAddress(addr string) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("address", addr)
		return nil
	}
}

func SetObjectName(objectName string) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("objectname", objectName)
		return nil
	}
}

func SetPrefixFilter(prefix string) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("prefix", prefix)
		return nil
	}
}

func SetPolicy(policy int) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("policy", policy)
		return nil
	}
}

func SetDataCount(dataCount int) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("datacount", dataCount)
		return nil
	}
}

func SetParityCount(parityCount int) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("paritycount", parityCount)
		return nil
	}
}

func NeedAvailTime(enabled bool) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("Avail", enabled)
		return nil
	}
}

func SetSecretKey(sk string) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("secretekey", sk)
		return nil
	}
}
func SetPassword(pwd string) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("password", pwd)
		return nil
	}
}

func ForceFlush(enabled bool) LfsOpts {
	return func(rb *RequestBuilder) error {
		rb.Option("force", enabled)
		return nil
	}
}

type PeerState struct {
	PeerID    string
	Connected bool
}

func (ps PeerState) String() string {
	if ps.Connected {
		return ps.PeerID + " connected"
	}
	return ps.PeerID + " unconnected"
}

type PeerList struct {
	Peers []PeerState
}

func (pl PeerList) String() string {
	var res string
	for _, ps := range pl.Peers {
		res += ps.String() + "\n"
	}
	return res
}

type QueryEventType int

const (
	SendingQuery QueryEventType = iota
	PeerResponse
	FinalPeer
	QueryError
	Provider
	Value
	AddingPeer
	DialingPeer
)

type QueryEvent struct {
	ID        peer.ID
	Type      QueryEventType
	Responses []*peer.AddrInfo
	Extra     string
}

type GetBlockResult struct {
	IsExist bool
}

type BlockStat struct {
	Key  string
	Size int
}

func (c Client) CreateUser(options ...LfsOpts) (*UserPrivMessage, error) {
	var user UserPrivMessage
	rb := c.Request("create")
	for _, option := range options {
		option(rb)
	}

	if err := rb.Exec(context.Background(), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c Client) StartUser(address string, options ...LfsOpts) error {
	var res StringList
	rb := c.Request("lfs/start", address)
	for _, option := range options {
		option(rb)
	}
	if err := rb.Exec(context.Background(), &res); err != nil {
		return err
	}
	return nil
}

func (c Client) Fsync(options ...LfsOpts) error {
	var res StringList
	rb := c.Request("lfs/fsync")
	for _, option := range options {
		option(rb)
	}

	if err := rb.Exec(context.Background(), &res); err != nil {
		return err
	}
	return nil
}

func (c Client) ShowStorage(options ...LfsOpts) error {
	var res string
	rb := c.Request("lfs/show_storage")
	for _, option := range options {
		option(rb)
	}

	if err := rb.Exec(context.Background(), &res); err != nil {
		return err
	}
	return nil
}

func (c Client) ListKeepers(options ...LfsOpts) (*PeerList, error) {
	var res *PeerList
	rb := c.Request("lfs/list_keepers")
	for _, option := range options {
		option(rb)
	}

	if err := rb.Exec(context.Background(), &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c Client) ChallengeTest(key, to string, options ...LfsOpts) (string, error) {
	var res string
	rb := c.Request("dht/challengeTest", key, to)
	for _, option := range options {
		option(rb)
	}

	if err := rb.Exec(context.Background(), &res); err != nil {
		return "", err
	}
	return res, nil
}

func (c Client) GetFrom(key, id string, options ...LfsOpts) (*QueryEvent, error) {
	var res *QueryEvent
	rb := c.Request("dht/getfrom", key, id)
	for _, option := range options {
		option(rb)
	}

	if err := rb.Exec(context.Background(), &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c Client) GetBlockFrom(key, id string, options ...LfsOpts) (string, error) {
	fmt.Println("in GetBlockFrom")
	var res string
	rb := c.Request("block/getfrom", key, id)
	for _, option := range options {
		option(rb)
	}

	if err := rb.Exec(context.Background(), &res); err != nil {
		return "", err
	}
	return res, nil
}
