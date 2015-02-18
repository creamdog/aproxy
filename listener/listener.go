package listener

import (
	"./http"
	nethttp "net/http"
)

type Listener interface {
	Start()
	Stop()
	IsRunning() bool
}

var Implementations = map[string]func(map[string]interface{}, func(map[string]interface{}, nethttp.ResponseWriter)) (Listener, error){
	"http": func(config map[string]interface{}, ondata func(map[string]interface{}, nethttp.ResponseWriter)) (Listener, error) {
		return http.Init(config, ondata)
	},
}
