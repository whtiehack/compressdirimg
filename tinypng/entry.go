package tinypng

import (
	"bytes"
	"io"
	"os"
	"sync"
)

func Compress(filePath string) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return compressOne(filePath, f)
}

// 数据库读写锁
var rwlock = new(sync.RWMutex)

func compressOne(name string, file io.Reader) ([]byte, error) {
	rawReader := file

	bufReader := new(bytes.Buffer)
	_, err := copyAndMd5(bufReader, rawReader)
	if err != nil {
		return nil, err
	}

	data := bufReader.Bytes()

	// 从网络中获取数据
	outBuf := new(bytes.Buffer)
	_, err = upload(bufReader, outBuf)
	if err != nil {
		return nil, err
	}
	data = outBuf.Bytes()
	// 返回
	return data, err
}
