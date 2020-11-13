package watcher

import (
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/fsnotify/fsnotify"
)

type Event struct {
	fsnotify.Op
	Path    string
	OldPath string
	os.FileInfo
}

// Watcher wrap for fsnotify
type Watcher struct {
	Event     chan Event
	Error     chan error
	done      chan struct{}
	fsWatcher *fsnotify.Watcher
}

// New create a new watcher
func New() *Watcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	return &Watcher{
		fsWatcher: watcher,
		Event:     make(chan Event, 50),
		Error:     make(chan error),
	}
}

// FilterOps filters which event op types should be returned
// when an event occurs.
func (w *Watcher) FilterOps(any interface{}) {
	// TODO
}

// AddRecursive adds either a single file or directory recursively to the file list.
func (w *Watcher) AddRecursive(dir string) error {
	dir = path.Clean(dir)
	rd, _ := ioutil.ReadDir(dir)
	err := w.fsWatcher.Add(dir)
	if err != nil {
		return err
	}
	log.Println("add watch dir", dir)
	for _, fi := range rd {
		if fi.IsDir() {
			w.AddRecursive(dir + "/" + fi.Name())
		}
	}
	return nil
}

// Start start watch
func (w *Watcher) Start(any interface{}) error {
	watcher := w.fsWatcher
	w.done = make(chan struct{})
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("watcher event", event)
				if event.Op&fsnotify.Create == fsnotify.Create {
					fi, err := os.Stat(event.Name)
					if err != nil {
						log.Println("get file stat failed", err, " event:", event)
						break
					}
					log.Println("create", fi.Name(), " event name", event.Name)
					if fi.IsDir() {
						log.Println("create a new dir,notify", event.Name)
						w.AddRecursive(event.Name)
						break
					}
					w.Event <- Event{
						Op:       event.Op,
						Path:     event.Name,
						OldPath:  event.Name,
						FileInfo: fi,
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				w.Error <- err
			}
		}
	}()
	<-w.done
	return nil
}

func (w *Watcher) close() error {
	close(w.Error)
	close(w.Event)
	close(w.done)
	return w.fsWatcher.Close()
}
