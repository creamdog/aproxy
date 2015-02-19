package main

import (
	"github.com/creamdog/aproxy/config"
	"github.com/creamdog/aproxy/config/file"
	"github.com/creamdog/aproxy/config/s3"
	"github.com/creamdog/aproxy/listener"
	"github.com/creamdog/aproxy/mappings"
	"github.com/creamdog/aproxy/cache"
	httppipe "github.com/creamdog/aproxy/pipes/http"
	//"log"
	"net/http"
	"time"	
	"github.com/creamdog/aproxy/log"
)

var mappingsCollection *mappings.Mappings
var cacheClient cache.CacheClient

const (
	defaultConfigFile = "config.json"
)

func main() {
	config, err := config.Load(defaultConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	c, err := cache.Get(config.Cache)
	if err != nil {
		log.Fatal(err)
	}
	cacheClient = c

	mappingsCollection = initializeMappings(config)
	listeners := initializeListeners(config)

	if section, exists := config.MappingRepo["s3"]; exists {
		s3.Start(mappingsCollection, section.(map[string]interface{}))
	} else {
		file.Start(mappingsCollection, "mapping-configuration")
	}

	for listeners.IsRunning() {
		time.Sleep(1 * time.Second)
	}
}

func ondata(data map[string]interface{}, w http.ResponseWriter) {

	mappings := mappingsCollection.Get()

	//log.Printf("mappings: %d, data: %v", len(*mappings), data)

	if requestMapping, err := mappings.GetMatch(data); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), 500)
	} else if requestMapping != nil {
		log.Printf("executing mapping: %v", requestMapping.Id)
		pipe := httppipe.New(cacheClient)
		pipe.Pipe(requestMapping, w)
	} else {
		http.Error(w, "these are not the droids you're looking for", 404)
		log.Printf("found no mapping matching request")
	}
}

type Listeners []listener.Listener

func (col Listeners) IsRunning() bool {
	for _, l := range col {
		if l.IsRunning() {
			return true
		}
	}
	return false
}

func initializeMappings(config *config.Config) *mappings.Mappings {
	mapping, err := mappings.Load(config.Mappings)
	if err != nil {
		log.Fatal(err)
	}
	return mapping
}

func initializeListeners(config *config.Config) Listeners {
	listeners := make([]listener.Listener, 0)
	for _, lconfig := range config.Listeners {
		if lconfig["type"] == nil {
			log.Fatalf("listener type property not set: %v", lconfig)
		}
		listener, err := listener.Implementations[lconfig["type"].(string)](lconfig, ondata)
		if err != nil {
			log.Fatal(err)
		}
		listeners = append(listeners, listener)
		listener.Start()
	}
	return listeners
}