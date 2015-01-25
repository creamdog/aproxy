package http

import(
	"log"
	"net/http"
	"strings"
	"fmt"
)

type HttpListener struct {
	Interface string
	UIPath string
	Started bool
	Mux *http.ServeMux
	OnData func(map[string]interface{}, http.ResponseWriter)
}

func Init(config map[string]interface{}, ondata func(map[string]interface{}, http.ResponseWriter)) (*HttpListener, error) {
	log.Printf("initialized http listener: %v", config)
	return &HttpListener{
		Interface : config["interface"].(string), 
		UIPath : config["ui"].(string), 
		Started: false,
		OnData: ondata,
		Mux: nil}, nil
}

func (listener *HttpListener) handle(w http.ResponseWriter, r *http.Request) {


	data := map[string]interface{}{
		"request" : map[string]interface{}{
			"method" : r.Method,
			"path" : r.URL.Path,
			"host" : r.Host,
			"protocol" : r.Proto,
			"uri" : r.RequestURI,
			"content-length" : fmt.Sprintf("%d", r.ContentLength),
		},
		"query" : map[string]interface{}{},
		"header" : map[string]interface{}{},
	}
	for key, values := range r.URL.Query() {
		if len(values) > 1 {
			data["query"].(map[string]interface{})[strings.ToLower(key)] = values
		} else {
			data["query"].(map[string]interface{})[strings.ToLower(key)] = values[0]
		}
	}
	for key, values := range r.Header {
		if len(values) > 1 {
			data["header"].(map[string]interface{})[strings.ToLower(key)] = values
		} else {
			data["header"].(map[string]interface{})[strings.ToLower(key)] = values[0]
		}
	}

	listener.OnData(data, w)
	log.Printf("%q\n", data)
}

func (listener *HttpListener) Start() {
	listener.Mux = http.NewServeMux()
	listener.Mux.Handle(listener.UIPath, http.StripPrefix(listener.UIPath, http.FileServer(http.Dir("http-files"))))
	listener.Mux.HandleFunc("/", listener.handle)
	go func() {
		err := http.ListenAndServe(listener.Interface, listener.Mux)
		if err != nil {
			listener.Mux = nil
			log.Fatal("ListenAndServe: ", err)
		}
	}()
	listener.Started = true
	log.Printf("started http listener %v\n", listener.Interface)
}

func (listener *HttpListener) Stop() {
	listener.Started = false
}

func (listener *HttpListener) IsRunning() bool {
	return listener.Started
}