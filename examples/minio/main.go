package main

// +build ignore

/*
 * mefs Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 mefs, Inc.
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

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"strconv"

	"github.com/memoio/mefs-sdk-go"
)

func main() {
	//预先准备好addr
	addr := "0xD60457e090e166305D3CEE0BCF3778C689B7441d"
	endpoint := "127.0.0.1:6711"
	mefsClient, err := mefs.New(endpoint, addr, "123456", true)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("连接成功")

	bucketname := "bucket01"
	err = mefsClient.MakeBucket(bucketname, "")
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("创建桶成功")
	//构造随机文件
	r := rand.Int63n(1024 * 1024 * 20)
	data := make([]byte, r)
	fillRandom(data)
	buf := bytes.NewBuffer(data)
	objectName := addr + "_" + strconv.Itoa(int(r))
	fmt.Println("  Begin to upload the ", objectName, "Size is", toStorageSize(r), "addr", addr)
	_, err = mefsClient.PutObject(bucketname, objectName, buf, r, mefs.PutObjectOptions{})
	p := path.Join(os.Getenv("GOPATH"), objectName)
	fmt.Println(p, err)
	err = mefsClient.FGetObject(bucketname, objectName, p, mefs.GetObjectOptions{})
	Obinfo, err := mefsClient.StatObject(bucketname, objectName, mefs.StatObjectOptions{})
	fmt.Println(Obinfo, err)
}

func fillRandom(p []byte) {
	for i := 0; i < len(p); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(p) && j < 7; j++ {
			p[i+j] = byte(val)
			val >>= 8
		}
	}
}

func toStorageSize(r int64) string {
	FloatStorage := float64(r)
	var OutStorage string
	if FloatStorage < 1024 && FloatStorage >= 0 {
		OutStorage = fmt.Sprintf("%.2f", FloatStorage) + "B"
	} else if FloatStorage < 1048576 && FloatStorage >= 1024 {
		OutStorage = fmt.Sprintf("%.2f", FloatStorage/1024) + "KB"
	} else if FloatStorage < 1073741824 && FloatStorage >= 1048576 {
		OutStorage = fmt.Sprintf("%.2f", FloatStorage/1048576) + "MB"
	} else {
		OutStorage = fmt.Sprintf("%.2f", FloatStorage/1073741824) + "GB"
	}
	return OutStorage
}
