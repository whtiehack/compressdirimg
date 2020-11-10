package tinypng

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

func TestUpload(t *testing.T) {
	t.Log(os.Getwd())
	reader, err := os.Open("../testdir/xx.jpg")
	if err != nil {
		t.Error(err)
		return
	}
	writer := bytes.NewBuffer(nil)
	r, err := upload(reader, writer)
	fmt.Println("r,", r, " err:", err, "rr:", len(writer.Bytes()))
}

func TestCompressOne(t *testing.T) {
	reader, _ := os.Open("../testdir/xx.jpg")
	ret, err := compressOne("ttt", reader, _db)
	if err != nil {
		t.Error("err", err)
		return
	}
	fmt.Println("ret:", len(ret))
}
