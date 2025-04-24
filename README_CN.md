# 适用于与MEFS的Go SDK 

mefs-sdk-go SDK提供了简单的API来访问MEFS的对象存储服务。


本文假设你已经有 [Go开发环境](https://golang.org/doc/install)。

## 从Github下载
```sh
go get -u github.com/memoio/mefs-sdk-go
```

## 初始化MinIO Client
MinIO client需要以下4个参数来连接与MEFS兼容的对象存储。

| 参数            | 描述                          |
| :-------------- | :---------------------------- |
| endpoint        | 对象存储服务的URL             |
| accessKeyID     | MEFS中的address               |
| secretAccessKey | 在该Gateway上的登录密码       |
| secure          | true代表使用HTTPS（暂未兼容） |


```go
package main

import (
	"github.com/memoio/mefs-sdk-go"
	"log"
)

func main() {
	//预先准备好addr
	addr := "0xD60457e090e166305D3CEE0BCF3778C689B7441d"
	endpoint := "127.0.0.1:6711"
	password := "123456"
	mefsClient, err := mefs.New(endpoint, addr, password, true)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("连接成功")
}
```

## 示例-文件上传
本示例连接到一个对象存储服务，创建一个存储桶并上传一个文件到存储桶中。


### FileUploader.go
```go
package main

import (
	"github.com/memoio/mefs-sdk-go"
	"log"
)

func main() {
	//预先准备好addr
	addr := "0xD60457e090e166305D3CEE0BCF3778C689B7441d"
	endpoint := "127.0.0.1:6711"
	useSSL := true

	// 初使化mefs client对象。
	mefsClient, err := mefs.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		log.Fatalln(err)
	}

	// 创建一个叫mymusic的存储桶。
	bucketName := "mymusic"

	err = mefsClient.MakeBucket(bucketName, location)
	if err != nil {
		// 检查存储桶是否已经存在。
		exists, err := mefsClient.BucketExists(bucketName)
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
	n, err := mefsClient.FPutObject(bucketName, objectName, filePath, minio.PutObjectOptions{ContentType:contentType})
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

### 支持API : 操作存储桶
* `MakeBucket`
* `ListBuckets`
* `BucketExists`
* `ListObjects`


### 支持API : 操作文件对象
* `FPutObject`
* `FGetObject`
* `FPutObjectWithContext`
* `FGetObjectWithContext`

### 支持API : 操作对象
* `GetObject`
* `PutObject`
* `GetObjectWithContext`
* `PutObjectWithContext`
* `PutObjectStreaming`
* `StatObject`

### 具体示例
* [示例](https://github.com/memoio/mefs-sdk-go/blob/master/examples/minio/main.go)
