package main

import (
	"compressDirImg/tinypng"
	"compressDirImg/watcher"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

func init() {
	log.SetOutput(os.Stdout)
}

const MaxFileSize = 5 * 1024 * 1024

func compressAllFile(pathname string) {
	rd, _ := ioutil.ReadDir(pathname)
	// ignore before 20 days
	zt := time.Now().AddDate(0, 0, -20)
	for _, fi := range rd {
		if fi.IsDir() {
			log.Printf("[%s]\n", pathname+"/"+fi.Name())
			compressAllFile(pathname + "/" + fi.Name())
		} else {
			log.Println(pathname, fi.Name())
			// ignore large file and small file.
			if fi.Size() > MaxFileSize || fi.Size() <= 10000 || fi.ModTime().Before(zt) {
				continue
			}
			p := pathname + "/" + fi.Name()
			err := tinypng.IsImage(p)
			if err != nil {
				log.Println("is not an image", p, err)
				continue
			}
			ret, err := tinypng.Compress(p)
			if err != nil {
				log.Println("ERRRR ", err)
				log.Println("ERRRR## ", err.Error() == "Unsupported media type")
			} else {
				err = ioutil.WriteFile(p, ret, os.ModePerm)
				if err != nil {
					log.Println("write file failed", p, err)
				}
			}
		}
	}
	return
}

func main() {
	dir := flag.String("dir", "", "set dir")
	skipCompressAll := flag.Bool("skipCompressAll", false, "skip compress all file")
	flag.Parse()
	if *dir == "" {
		log.Fatalln("no dir")
	}
	log.Println("watch dir:", *dir)
	if !*skipCompressAll {
		compressAllFile(*dir)
	}
	w := watcher.New()
	// Only notify rename and move events.
	w.FilterOps(nil)
	go func() {
		m := map[string]bool{}
		for {
			select {
			case event, ok := <-w.Event:
				if !ok {
					return
				}
				go processEvent(event, m)
			case err, ok := <-w.Error:
				if !ok {
					return
				}
				log.Fatalln(err)
			}
		}
	}()

	if err := w.AddRecursive(*dir); err != nil {
		log.Fatalln(err)
	}
	log.Println("start watching")
	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}

var mapMutex sync.Mutex

func processEvent(event watcher.Event, m map[string]bool) {
	mapMutex.Lock()
	defer mapMutex.Unlock()
	// keep to check file write ended
	time.Sleep(1 * time.Second)
	log.Println("path", event.Path, "name", event.Name(), "size", lenReadable(int(event.Size()), 2)) // Print the event's info.
	if !event.FileInfo.IsDir() && !m[event.Path] &&
		event.FileInfo.Size() < MaxFileSize && event.FileInfo.Size() > 10000 {
		err := tinypng.IsImage(event.Path)
		if err != nil {
			log.Println("is not an image?", event.Path, err)
			return
		}
		// 检查文件类型
		m[event.Path] = true
		uploadProcess(event)
	} else {
		log.Println("file created:", event)
	}
	delete(m, event.Path)
}

func uploadProcess(event watcher.Event) {
	tryCount := 0
	for {
		ret, err := tinypng.Compress(event.Path)
		if err != nil {
			log.Println("compress err", event.Path, err)
			if err.Error() == "Unsupported media type" {
				break
			}
			if tryCount > 4 {
				log.Println("try count > 4", event.Path)
				break
			} else if tryCount == 1 {
				tinypng.UseBackAddress = true
			}
			time.Sleep(30 * time.Second)
			tryCount++
			continue
		}
		tinypng.UseBackAddress = false
		fi, err := os.Stat(event.Path)
		if err == nil {
			event.FileInfo = fi
		}
		log.Println("file compress success", event.Path,
			"size", lenReadable(int(event.FileInfo.Size()), 2),
			"new size", lenReadable(len(ret), 2),
			err,
		)
		err = ioutil.WriteFile(event.Path, ret, os.ModePerm)
		break
	}
}

const (
	TB = 1000000000000
	GB = 1000000000
	MB = 1000000
	KB = 1000
)

func lenReadable(length int, decimals int) (out string) {
	var unit string
	var i int
	var remainder int

	// Get whole number, and the remainder for decimals
	if length > TB {
		unit = "TB"
		i = length / TB
		remainder = length - (i * TB)
	} else if length > GB {
		unit = "GB"
		i = length / GB
		remainder = length - (i * GB)
	} else if length > MB {
		unit = "MB"
		i = length / MB
		remainder = length - (i * MB)
	} else if length > KB {
		unit = "KB"
		i = length / KB
		remainder = length - (i * KB)
	} else {
		return strconv.Itoa(length) + " B"
	}

	if decimals == 0 {
		return strconv.Itoa(i) + " " + unit
	}

	// This is to calculate missing leading zeroes
	width := 0
	if remainder > GB {
		width = 12
	} else if remainder > MB {
		width = 9
	} else if remainder > KB {
		width = 6
	} else {
		width = 3
	}

	// Insert missing leading zeroes
	remainderString := strconv.Itoa(remainder)
	for iter := len(remainderString); iter < width; iter++ {
		remainderString = "0" + remainderString
	}
	if decimals > len(remainderString) {
		decimals = len(remainderString)
	}

	return fmt.Sprintf("%d.%s %s", i, remainderString[:decimals], unit)
}
