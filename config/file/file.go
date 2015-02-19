package file

import (
	"../../mappings"
	"encoding/json"
	"io/ioutil"
	"log"
	"path"
	"sync"
	"time"
)

type Listener struct {
	Seen    map[string]time.Time
	Lock    *sync.Mutex
	Mapping *mappings.Mappings
	Path    string
}

func Start(mapping *mappings.Mappings, path string) {
	l := &Listener{make(map[string]time.Time, 0), &sync.Mutex{}, mapping, path}
	go l.poll()
}

func (listener *Listener) poll() {
	for {
		log.Printf("polling %v", listener.Path)
		files, _ := ioutil.ReadDir(listener.Path)
		for _, f := range files {
			fpath := path.Join(listener.Path, f.Name())
			if f.IsDir() {
				continue
			}
			if modTime, exist := listener.Seen[fpath]; exist && modTime == f.ModTime() {
				continue
			}
			listener.Seen[fpath] = f.ModTime()
			listener.loadFile(fpath)
		}
		time.Sleep(1 * time.Second)
	}
}

func (listener *Listener) loadFile(filename string) {
	log.Printf("loading file %v", filename)
	if bytes, err := ioutil.ReadFile(filename); err != nil {
		return
	} else {
		var config map[string]interface{}
		if err = json.Unmarshal(bytes, &config); err != nil {
			return
		} else {
			listener.Mapping.Register(config["mappings"].(map[string]interface{}))
		}
	}
}
