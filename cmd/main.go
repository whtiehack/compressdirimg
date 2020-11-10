package main

import (
	"compressDirImg/tinypng"
	"flag"
	"github.com/radovskyb/watcher"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

func init() {
	log.SetOutput(os.Stdout)
}

const MaxFileSize = 5 * 1024 * 1024

func compressAllFile(pathname string) {
	rd, _ := ioutil.ReadDir(pathname)
	for _, fi := range rd {
		if fi.IsDir() {
			log.Printf("[%s]\n", pathname+"/"+fi.Name())
			compressAllFile(pathname + "/" + fi.Name())
		} else {
			log.Println(pathname, fi.Name())
			if fi.Size() > MaxFileSize {
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
	flag.Parse()
	if *dir == "" {
		log.Fatalln("no dir")
	}
	log.Println("watch dir:", *dir)
	compressAllFile(*dir)
	w := watcher.New()
	// Only notify rename and move events.
	w.FilterOps(watcher.Create)
	go func() {
		m := map[string]bool{}
		for {
			select {
			case event := <-w.Event:
				processEvent(event, m)
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	if err := w.AddRecursive(*dir); err != nil {
		log.Fatalln(err)
	}
	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}

var mapMutex sync.Mutex

func processEvent(event watcher.Event, m map[string]bool) {
	mapMutex.Lock()
	defer mapMutex.Unlock()
	log.Println("path", event.Path, "name", event.Name(), "fileinfo", event.FileInfo) // Print the event's info.
	if !event.FileInfo.IsDir() && !m[event.Path] &&
		event.FileInfo.Size() < MaxFileSize && event.FileInfo.Size() > 10000 {
		err := tinypng.IsImage(event.Path)
		if err != nil {
			log.Println("is not an image", event.Path, err)
			return
		}
		// 检查文件类型
		m[event.Path] = true
		go uploadProcess(event, m)
	} else {
		log.Println("file created:", event)
	}
}

func uploadProcess(event watcher.Event, m map[string]bool) {
	for {
		ret, err := tinypng.Compress(event.Path)
		if err != nil {
			log.Println("compress err", err)
			time.Sleep(30 * time.Second)
			continue
		}
		err = ioutil.WriteFile(event.Path, ret, os.ModePerm)
		break
	}
	log.Println("file compress success", event.Path)
	mapMutex.Lock()
	defer mapMutex.Unlock()
	delete(m, event.Path)
}
