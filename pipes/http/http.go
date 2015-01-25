package http

import(
	"github.com/creamdog/aproxy/mappings"
	"io"
	"net/http"
	"strings"
	"log"
)

type HttpPipe struct {

}

func New() *HttpPipe {
	return &HttpPipe{}
}

func (pipe *HttpPipe) Pipe(mapping *mappings.RequestMapping, w http.ResponseWriter) {
	request, err := http.NewRequest(mapping.Verb, mapping.Uri, strings.NewReader(mapping.Body))

	log.Printf("%q %q, %q", mapping.Verb, mapping.Uri, mapping.Headers)
	if err != nil {
		http.Error(w, err.Error(), 503)
	}
	request.ContentLength = int64(len(mapping.Body))
	for key, value := range mapping.Headers {
		request.Header[key] = []string{value}
	}
	client := &http.Client{}
	if response, err := client.Do(request); err != nil {
		http.Error(w, err.Error(), 500)
	} else {
		for key, value := range response.Header {
			w.Header().Set(key, value[0])
		}
		//w.Header().Set("fsfsdf", "kfjdkfjsdh")
		w.WriteHeader(response.StatusCode)
		defer response.Body.Close()
		io.Copy(w, response.Body)
	}
}