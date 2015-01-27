package http

import (
	"github.com/creamdog/aproxy/mappings"
	"io"
	"log"
	"net/http"
	"strings"
	"encoding/json"
	"bytes"
	"fmt"
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

	log.Printf("request.ContentLength: %d, mapping.Body: %v", request.ContentLength, mapping.Body)

	for key, value := range mapping.Headers {
		request.Header[key] = []string{value}
	}

	request.Header["Transfer-Encoding"] = []string{""}

	client := &http.Client{}
	if response, err := client.Do(request); err != nil {
		http.Error(w, err.Error(), 500)
	} else {

		defer response.Body.Close()
		_, notransform := (*mapping.Data)["query"].(map[string]interface{})["_notransform"]
		if mapping.CompiledTransform != nil && !notransform {
			log.Printf("response.ContentLength: %d", response.ContentLength)
			readBuffer := make([]byte, 1024)
			buffer := make([]byte, 0)
			read := true
			max := 1 * 1024 * 1024
			for read {

				if len(buffer) >= max {
					http.Error(w, fmt.Errorf("transform: maximum input site exceeded %d bytes", max).Error(), 500)
					return
				}

				read = false
				num, err := response.Body.Read(readBuffer)
				if err != nil {
					if err.Error() == "EOF" {
						buffer = append(buffer, readBuffer[:num]...)
						break
					}
					http.Error(w, err.Error(), 500)
					return
				}
				log.Printf("read %d bytes", num)
				if num > 0 {
					buffer = append(buffer, readBuffer[:num]...)
					read = true
				}
			}

			log.Printf("buffer[%d]: %v", len(buffer), string(buffer))

			responseData := map[string]interface{}{}

			if mapping.Mapping.Target.Transform.Type == "json" {
				if err := json.Unmarshal(buffer, &responseData); err != nil {
					http.Error(w, err.Error() + " : " + string(buffer), 500)
					return
				}
			} else if mapping.Mapping.Target.Transform.Type == "regexp" {
				re := mapping.Mapping.Target.Transform.Regexp.FindStringSubmatch(string(buffer))
				names := mapping.Mapping.Target.Transform.Regexp.SubexpNames()
				if re != nil {
					for i, n := range re {
						if len(names[i]) > 0 {
							responseData[names[i]] = n
						}
					}
				}
			}

			log.Printf("responseData: %v", responseData)

			data := map[string]interface{}{
				"data" : responseData,
			}
			for key, value := range *mapping.Data {
				data[key] = value
			}

			var renderBuffer bytes.Buffer
			if err := mapping.CompiledTransform.Execute(&renderBuffer, data); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			for key, value := range response.Header {
				w.Header().Set(key, value[0])
			}
			w.WriteHeader(response.StatusCode)
			fmt.Fprint(w, renderBuffer.String())
			return
		} 

		for key, value := range response.Header {
			w.Header().Set(key, value[0])
		}
		w.WriteHeader(response.StatusCode)
		io.Copy(w, response.Body)
	}
}
