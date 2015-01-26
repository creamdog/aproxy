package main

import (
	"github.com/creamdog/aproxy/config/file"
	"github.com/creamdog/aproxy/config/s3"
	"github.com/creamdog/aproxy/listener"
	"github.com/creamdog/aproxy/mappings"
	httppipe "github.com/creamdog/aproxy/pipes/http"
	"log"
	"net/http"
	"time"
)

var mappingsCollection *mappings.Mappings

func main() {
	config, err := LoadConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

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

	log.Printf("%q", len(*mappingsCollection))

	if requestMapping, err := mappingsCollection.Get().GetMatch(data); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), 500)
	} else if requestMapping != nil {
		log.Printf("executing mapping: %v", requestMapping.Id)
		pipe := httppipe.New()
		pipe.Pipe(requestMapping, w)
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

func initializeMappings(config *Config) *mappings.Mappings {
	mapping, err := mappings.Load(config.Mappings)
	if err != nil {
		log.Fatal(err)
	}
	return mapping
}

func initializeListeners(config *Config) Listeners {
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

/*

- caching / cache strategy /

*/
