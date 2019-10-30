/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 MinIO, Inc.
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
	"context"
	"net/http"
	"time"

	"github.com/memoio/minio-go/pkg/s3utils"
)

// BucketExists verify if bucket exists and you have permission to access it.
func (c Client) BucketExists(bucketName string) (bool, error) {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return false, err
	}

	var bks Buckets
	rb := c.Request("lfs/head_Bucket", bucketName)
	creds, err := c.credsProvider.Get()
	if err != nil {
		return false, err
	}
	rb.Option("address", creds.AccessKeyID)

	if err := rb.Exec(context.Background(), &bks); err != nil {
		return false, err
	}
	return true, nil
}

// List of header keys to be filtered, usually
// from all S3 API http responses.
var defaultFilterKeys = []string{
	"Connection",
	"Transfer-Encoding",
	"Accept-Ranges",
	"Date",
	"Server",
	"Vary",
	"x-amz-bucket-region",
	"x-amz-request-id",
	"x-amz-id-2",
	"Content-Security-Policy",
	"X-Xss-Protection",

	// Add new headers to be ignored.
}

// Extract only necessary metadata header key/values by
// filtering them out with a list of custom header keys.
func extractObjMetadata(header http.Header) http.Header {
	filterKeys := append([]string{
		"ETag",
		"Content-Length",
		"Last-Modified",
		"Content-Type",
		"Expires",
	}, defaultFilterKeys...)
	return filterHeader(header, filterKeys)
}

// StatObject verifies if object exists and you have permission to access.
func (c Client) StatObject(bucketName, objectName string, opts StatObjectOptions) (ObjectInfo, error) {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return ObjectInfo{}, err
	}
	if err := s3utils.CheckValidObjectName(objectName); err != nil {
		return ObjectInfo{}, err
	}
	return c.statObject(context.Background(), bucketName, objectName, opts)
}

// Lower level API for statObject supporting pre-conditions and range headers.
func (c Client) statObject(ctx context.Context, bucketName, objectName string, opts StatObjectOptions) (ObjectInfo, error) {
	// Input validation.
	if err := s3utils.CheckValidBucketName(bucketName); err != nil {
		return ObjectInfo{}, err
	}
	if err := s3utils.CheckValidObjectName(objectName); err != nil {
		return ObjectInfo{}, err
	}

	rb := c.Request("lfs/head_object", bucketName, objectName)
	creds, err := c.credsProvider.Get()
	if err != nil {
		return ObjectInfo{}, err
	}
	rb.Option("address", creds.AccessKeyID)
	var objs Objects
	if err := rb.Exec(ctx, &objs); err != nil {
		return ObjectInfo{}, err
	}

	t, _ := time.Parse(SHOWTIME, objs.Objects[0].Ctime)
	return ObjectInfo{
		ETag:         objs.Objects[0].MD5,
		Key:          objs.Objects[0].ObjectName,
		Size:         int64(objs.Objects[0].ObjectSize),
		LastModified: t,
	}, nil
}
