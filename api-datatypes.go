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

package mefs

import (
	"net/http"
	"time"
)

//SHOWTIME 用于输出给使用者
const SHOWTIME = "2006-01-02 Mon 15:04:05 MST"

// BucketInfo container for bucket metadata.
type BucketInfo struct {
	// The name of the bucket.
	Name string `json:"name"`
	// Date the bucket was created.
	CreationDate time.Time `json:"creationDate"`
}

// ObjectInfo container for object metadata.
type ObjectInfo struct {
	// An ETag is optionally set to md5sum of an object.  In case of multipart objects,
	// ETag is of the form MD5SUM-N where MD5SUM is md5sum of all individual md5sums of
	// each parts concatenated into one string.
	ETag string `json:"etag"`

	Key          string    `json:"name"`         // Name of the object
	LastModified time.Time `json:"lastModified"` // Date and time the object was last modified.
	Size         int64     `json:"size"`         // Size in bytes of the object.
	ContentType  string    `json:"contentType"`  // A standard MIME type describing the format of the object data.
	Expires      time.Time `json:"expires"`      // The date and time at which the object is no longer able to be cached.

	// Collection of additional metadata on the object.
	// eg: x-amz-meta-*, content-encoding etc.
	Metadata http.Header `json:"metadata" xml:"-"`

	// Owner name.
	Owner struct {
		DisplayName string `json:"name"`
		ID          string `json:"id"`
	} `json:"owner"`

	// The class of storage used to store the object.
	StorageClass string `json:"storageClass"`

	// Error
	Err error `json:"-"`
}

// ObjectMultipartInfo container for multipart object metadata.
type ObjectMultipartInfo struct {
	// Date and time at which the multipart upload was initiated.
	Initiated time.Time `type:"timestamp" timestampFormat:"iso8601"`

	Initiator initiator
	Owner     owner

	// The type of storage to use for the object. Defaults to 'STANDARD'.
	StorageClass string

	// Key of the object for which the multipart upload was initiated.
	Key string

	// Size in bytes of the object.
	Size int64

	// Upload ID that identifies the multipart upload.
	UploadID string `xml:"UploadId"`

	// Error
	Err error
}

type ObjectStat struct {
	ObjectName     string
	ObjectSize     int32
	MD5            string
	Ctime          string
	Dir            bool
	LatestChalTime string
}

type Objects struct {
	Method  string
	Objects []ObjectStat
}

type BucketStat struct {
	BucketName  string
	BucketID    int32
	Ctime       string
	Policy      int32
	DataCount   int32
	ParityCount int32
}

type Buckets struct {
	Method  string
	Buckets []BucketStat
}
