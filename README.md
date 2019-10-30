# 适用于与Amazon S3兼容云存储的MinIO Go SDK [![Slack](https://slack.min.io/slack?type=svg)](https://slack.min.io) [![Sourcegraph](https://sourcegraph.com/github.com/memoio/minio-go/-/badge.svg)](https://sourcegraph.com/github.com/memoio/minio-go?badge)

MinIO Go Client SDK提供了简单的API来访问MEFS的对象存储服务。


本文假设你已经有 [Go开发环境](https://golang.org/doc/install)。

## 从Github下载
```sh
go get -u github.com/memoio/minio-go
```

## 初始化MinIO Client
MinIO client需要以下4个参数来连接与MEFS兼容的对象存储。

| 参数            | 描述                          |
| :-------------- | :---------------------------- |
| endpoint        | 对象存储服务的URL             |
| accessKeyID     | MEFS中的address               |
| secretAccessKey | 在该gateway上的登录密码       |
| secure          | true代表使用HTTPS（暂未兼容） |


```go
func main() {
	//预先准备好addr
	addr := "0xD60457e090e166305D3CEE0BCF3778C689B7441d"
	endpoint := "127.0.0.1:6711"
	password := "123456"
	minioClient, err := minio.New(endpoint, addr, password, true)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("连接成功")
}
```

## 示例-文件上传
本示例连接到一个对象存储服务，创建一个存储桶并上传一个文件到存储桶中。

我们在本示例中使用运行在 [https://play.min.io](https://play.min.io) 上的MinIO服务，你可以用这个服务来开发和测试。示例中的访问凭据是公开的。

### FileUploader.go
```go
package main

import (
	"github.com/memoio/minio-go"
	"log"
)

func main() {
	//预先准备好addr
	addr := "0xD60457e090e166305D3CEE0BCF3778C689B7441d"
	endpoint := "127.0.0.1:6711"
	useSSL := true

	// 初使化minio client对象。
	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		log.Fatalln(err)
	}

	// 创建一个叫mymusic的存储桶。
	bucketName := "mymusic"

	err = minioClient.MakeBucket(bucketName, location)
	if err != nil {
		// 检查存储桶是否已经存在。
		exists, err := minioClient.BucketExists(bucketName)
		if err == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			log.Fatalln(err)
		}
	}
	log.Printf("Successfully created %s\n", bucketName)

	// 上传一个zip文件。
	objectName := "golden-oldies.zip"
	filePath := "/tmp/golden-oldies.zip"
	contentType := "application/zip"

	// 使用FPutObject上传一个zip文件。
	n, err := minioClient.FPutObject(bucketName, objectName, filePath, minio.PutObjectOptions{ContentType:contentType})
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Successfully uploaded %s of size %d\n", objectName, n)
}
```

### 运行FileUploader
```sh
go run file-uploader.go
2016/08/13 17:03:28 Successfully created mymusic 
2016/08/13 17:03:40 Successfully uploaded golden-oldies.zip of size 16253413

mc ls play/mymusic/
[2016-05-27 16:02:16 PDT]  17MiB golden-oldies.zip
```

## API文档
完整的API文档在这里。
* [完整API文档](https://docs.min.io/docs/golang-client-api-reference)

### API文档 : 操作存储桶
* [`MakeBucket`](https://docs.min.io/docs/golang-client-api-reference#MakeBucket)
* [`ListBuckets`](https://docs.min.io/docs/golang-client-api-reference#ListBuckets)
* [`BucketExists`](https://docs.min.io/docs/golang-client-api-reference#BucketExists)
* [`ListObjects`](https://docs.min.io/docs/golang-client-api-reference#ListObjects)


### API文档 : 操作文件对象
* [`FPutObject`](https://docs.min.io/docs/golang-client-api-reference#FPutObject)
* [`FGetObject`](https://docs.min.io/docs/golang-client-api-reference#FPutObject)
* [`FPutObjectWithContext`](https://docs.min.io/docs/golang-client-api-reference#FPutObjectWithContext)
* [`FGetObjectWithContext`](https://docs.min.io/docs/golang-client-api-reference#FGetObjectWithContext)

### API文档 : 操作对象
* [`GetObject`](https://docs.min.io/docs/golang-client-api-reference#GetObject)
* [`PutObject`](https://docs.min.io/docs/golang-client-api-reference#PutObject)
* [`GetObjectWithContext`](https://docs.min.io/docs/golang-client-api-reference#GetObjectWithContext)
* [`PutObjectWithContext`](https://docs.min.io/docs/golang-client-api-reference#PutObjectWithContext)
* [`PutObjectStreaming`](https://docs.min.io/docs/golang-client-api-reference#PutObjectStreaming)
* [`StatObject`](https://docs.min.io/docs/golang-client-api-reference#StatObject)

### 具体示例
* [示例](https://github.com/memoio/minio-go/blob/master/examples/minio/main.go)
