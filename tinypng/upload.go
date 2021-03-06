package tinypng

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"sync"
	"time"
)

func newCookie() *cookiejar.Jar {
	c, _ := cookiejar.New(nil)
	return c
}

var client = &http.Client{
	Timeout: 5 * time.Minute, // timeout -> 30秒
	Jar:     newCookie(),
}
var downloadClient = &http.Client{
	Timeout: 10 * time.Minute, // timeout -> 30秒
	Jar:     newCookie(),
}

type tinyPngResult struct {
	Input struct {
		Size int64  `json:"size"`
		Type string `json:"type"`
	} `json:"input"`
	Output struct {
		Height int64   `json:"height"`
		Ratio  float64 `json:"ratio"`
		Size   int64   `json:"size"`
		Type   string  `json:"type"`
		URL    string  `json:"url"`
		Width  int64   `json:"width"`
	} `json:"output"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

var uploadMutx sync.Mutex

var prevTime time.Time

var defaultAddress = "https://tinypng.com"
var backAddress = ""

var UseBackAddress = false

func init() {
	backAddress = os.Getenv("TINY_PNG_BACKUP")
}

func getNowAddress() string {
	if UseBackAddress && backAddress != "" {
		return backAddress
	}
	return defaultAddress
}

func upload(r io.Reader, w io.Writer) (string, error) {
	uploadMutx.Lock()
	uploadMutx.Unlock()
	sub := time.Now().Sub(prevTime)
	if sub < 8*time.Second {
		time.Sleep(8*time.Second - sub)
	}
	prevTime = time.Now()
	if r == nil {
		return "", errors.New("upload() can't read from nil io.Reader")
	}

	if w == nil {
		return "", errors.New("upload() can't write to nil io.Writer")
	}

	var uploadResp *http.Response

	defer func() {
		if uploadResp != nil {
			uploadResp.Body.Close()
		}
	}()

	// 上传图片文件
	post, err := http.NewRequest("POST", getNowAddress()+"/web/shrink", r)
	if err != nil {
		return "", err
	}
	post.Header.Set("Accept", "*/*")
	post.Header.Set("Accept-Encoding", "gzip, deflate")
	post.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	post.Header.Set("Referer", "https://tinypng.com/")
	post.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1.1 Safari/605.1.15")

	uploadResp, err = client.Do(post)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(uploadResp.Body)
	if err != nil {
		return "", err
	}
	// {"input":{"size":128321,"type":"image/jpeg"},"output":{"size":30316,"type":"image/jpeg","width":889,
	//"height":1028,"ratio":0.2363,"url":"https://tinypng.com/web/output/qyyx4nbhy0r09y5058ee6f6qv21e4nqg"}}
	data := new(tinyPngResult)
	log.Println("upload result:", string(body))
	err = json.Unmarshal(body, data)
	if err != nil {
		return string(body), err
	}
	if data.Error == "Unsupported media type" {
		return "", errors.New(data.Error)
	}
	if data.Output.URL == "" {
		return string(body), errors.New("result without url..~~")
	}
	tmpDownloadUrl := data.Output.URL
	tmpDownloadUrl = strings.Replace(tmpDownloadUrl, defaultAddress, getNowAddress(), 1)
	getRequest, err := http.NewRequest("GET", tmpDownloadUrl, nil)
	if err != nil {
		return "", err
	}
	getRequest.Header.Set("Accept", "*/*")
	getRequest.Header.Set("Accept-Encoding", "gzip, deflate")
	getRequest.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	getRequest.Header.Set("Referer", "https://tinypng.com/")
	getRequest.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1.1 Safari/605.1.15")

	downloadRet, err := downloadClient.Do(getRequest)
	if err != nil {
		return string(body), err
	}
	defer downloadRet.Body.Close()
	log.Println("download success", data.Output.URL)
	return copyAndMd5(w, downloadRet.Body)
}
