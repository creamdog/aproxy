package mappings

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"io"
)

type Mapping struct {
	Id      string
	Target  *TargetMapping
	Mapping map[string][]string
	Caching *CacheStrategy
}

type CacheStrategy struct {
	Key string
	Seconds int
}

type TargetMapping struct {
	Headers map[string]string
	Verb    string
	Body    string
	Uri     string
	Stub	bool
	Transform *TargetTransform
}
type TargetTransform struct {
	Type string
	Regexp *regexp.Regexp
	Template string
	Headers map[string]string
}

func (q *Mapping) Compile() (*CompiledMapping, error) {
	body, err := template.New(q.Id + "_body").Parse(q.Target.Body)
	if err != nil {
		return nil, err
	}
	url, err := template.New(q.Id + "_url").Parse(q.Target.Uri)
	if err != nil {
		return nil, err
	}

	var transform *template.Template
	if q.Target.Transform != nil {
		transform, err = template.New(q.Id + "_transform").Parse(q.Target.Transform.Template)
		if err != nil {
			return nil, err
		}
		log.Printf("%v => compiled target transform: %v", q.Id, q.Target.Transform.Template)
	}

	var cacheKey *template.Template
	if q.Caching != nil {
		q.Caching.Key = q.Id + ":" + q.Caching.Key
		cacheKey, err = template.New(q.Id + "_cachekey").Parse(q.Caching.Key)
		if err != nil {
			return nil, err
		}
		log.Printf("%v => compiled cache key: %v", q.Id, q.Caching.Key)
	}

	compiledMappings := map[string][]*regexp.Regexp{}
	for key, values := range q.Mapping {
		for _, value := range values {
			compiledRegexp, err := regexp.Compile(value)
			if err != nil {
				return nil, err
			}
			if _, exists := compiledMappings[key]; !exists {
				compiledMappings[key] = make([]*regexp.Regexp, 0)
			}
			compiledMappings[key] = append(compiledMappings[key], compiledRegexp)
			log.Printf("compiled mapping %v[%v](%v)\n", q.Id, key, value)
		}
	}

	return &CompiledMapping{
		Mapping:         q,
		CompiledBody:    body,
		CompiledUrl:     url,
		CompiledTransform: transform,
		CompiledMapping: compiledMappings,
		CompiledCacheKey: cacheKey,
	}, nil
}

type CompiledMapping struct {
	Mapping         *Mapping
	CompiledBody    *template.Template
	CompiledUrl     *template.Template
	CompiledTransform     *template.Template
	CompiledCacheKey *template.Template
	CompiledMapping map[string][]*regexp.Regexp
}

type RequestMapping struct {
	Id      string
	Body    string
	Headers map[string]string
	Verb    string
	Uri     string
	Mapping *Mapping
	CompiledTransform *template.Template
	Data    *map[string]interface{}
	CacheKey string
	RequestStream io.ReadCloser
}

func (cm *CompiledMapping) Prepare(data map[string]interface{}) (*RequestMapping, error) {
	body, err := cm.Body(data)
	if err != nil {
		return nil, err
	}
	uri, err := cm.Uri(data)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{}
	for key, value := range cm.Mapping.Target.Headers {
		if len(value) > 0 {
			headers[key] = value
		} else if headerdata, exists := data["header"].(map[string]interface{}); exists {
			for k, value := range headerdata {
				if strings.ToLower(k) == strings.ToLower(key) {
					if str, isString := value.(string); isString {
						headers[key] = str
					} else if strArray, isStringArray := value.([]string); isStringArray {
						headers[key] = strArray[0]
					}
				}
			}
		}
	}

	cachekey := ""
	if cm.CompiledCacheKey != nil {
		var buffer bytes.Buffer
		if err := cm.CompiledCacheKey.Execute(&buffer, data); err != nil {
			log.Printf("unable to transform cache key", err)
		}
		cachekey = buffer.String()
		log.Printf("transformed cache key: %s", cachekey)
	}

	return &RequestMapping{
		Id:      cm.Mapping.Id,
		Body:    body,
		Headers: headers,
		Verb:    cm.Mapping.Target.Verb,
		Uri:     uri,
		Mapping: cm.Mapping,
		CompiledTransform: cm.CompiledTransform,
		Data: &data,
		RequestStream: data["request"].(map[string]interface{})["body"].(io.ReadCloser),
		CacheKey: cachekey,
	}, nil
}

func (cm *CompiledMapping) Body(data map[string]interface{}) (string, error) {
	var buffer bytes.Buffer
	if err := cm.CompiledBody.Execute(&buffer, data); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func (cm *CompiledMapping) Uri(data map[string]interface{}) (string, error) {
	var buffer bytes.Buffer
	if err := cm.CompiledUrl.Execute(&buffer, data); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

type Mappings []*CompiledMapping

func flatten(keypath string, d map[string]interface{}) map[string]interface{} {
	if len(keypath) > 0 {
		keypath = keypath + "."
	}
	tmp := map[string]interface{}{}
	for key, value := range d {
		if m, ok := value.(map[string]interface{}); ok {
			for k, v := range flatten(keypath+key, m) {
				tmp[k] = v
			}
		} else if values, ok := value.([]string); ok {
			tmp[keypath+key] = values
		} else {
			tmp[keypath+key] = value
		}
	}
	return tmp
}

func (m Mappings) GetMatch(complexData map[string]interface{}) (*RequestMapping, error) {
	data := flatten("", complexData)
	for _, cm := range m {
		isMatch := true
		for key, regexpList := range cm.CompiledMapping {
			//log.Printf("checking %q, key: %v, data: %q", m, key, data)
			if value, exists := data[key]; !exists {
				isMatch = false
				break
			} else {
				for _, regexp := range regexpList {
					if values, ok := value.([]string); ok {
						for _, value := range values {
							//log.Printf("checking VALUE %q, key: %v, value: %v", m, key, value)
							if regexp.Match([]byte(value)) {
								break
							}
							isMatch = false
						}
					} else if value, ok := value.(string); ok {
						//log.Printf("checking VALUE %q, key: %v, value: %v", m, key, value)
						if !regexp.Match([]byte(value)) {
							isMatch = false
							break
						}
					}
				}
			}
		}
		if isMatch {
			log.Printf("matched")
			return cm.Prepare(complexData)
		}
	}
	return nil, nil
}

var registerMutex *sync.Mutex = &sync.Mutex{}

func (list *Mappings) Get() *Mappings {
	registerMutex.Lock()
	defer registerMutex.Unlock()
	clone := make(Mappings, 0)
	for _, value := range *list {
		clone = append(clone, value)
	}
	return &clone
}

func (list *Mappings) DeRegister(ids []string) {
	registerMutex.Lock()
	defer registerMutex.Unlock()
	for _, id := range ids {
		deleted := true
		for deleted {
			deleted = false
			for i, value := range *list {
				if value.Mapping.Id == id {
					log.Printf("deleting %v", id)
					*list = (*list)[:i+copy((*list)[i:], (*list)[i+1:])]
					deleted = true
					break
				}
			}
		}
	}
}

func (list *Mappings) Register(config map[string]interface{}) ([]string, error) {

	registerMutex.Lock()
	defer registerMutex.Unlock()

	loadedIds := make([]string, 0)

	for id, data := range config {

		deleted := true
		for deleted {
			deleted = false
			for i, value := range *list {
				if value.Mapping.Id == id {
					log.Printf("deleting %v", id)
					*list = (*list)[:i+copy((*list)[i:], (*list)[i+1:])]
					deleted = true
					break
				}
			}
		}

		transform, err := parseTargetTransform(data.(map[string]interface{})["target"].(map[string]interface{})["transform"])
		if err != nil {
			return nil, err
		}

		var cache *CacheStrategy

		if cacheConfig, exists := data.(map[string]interface{})["cache_strategy"]; exists {
			cache = &CacheStrategy{
				Key : cacheConfig.(map[string]interface{})["key"].(string),
				Seconds : int(cacheConfig.(map[string]interface{})["duration_seconds"].(float64)),
			}
		}

		log.Printf("cache: %v", cache)

		m := &Mapping{
			Id: id,
			Target: &TargetMapping{
				Headers: func() map[string]string {
					tmp := data.(map[string]interface{})["target"].(map[string]interface{})["headers"].(map[string]interface{})
					list := map[string]string{}
					for key, v := range tmp {
						list[key] = v.(string)
					}
					return list
				}(),
				Verb: strOrEmpty(data.(map[string]interface{})["target"].(map[string]interface{})["verb"]),
				Stub: boolOrFalse(data.(map[string]interface{})["target"].(map[string]interface{})["stub"]),
				Body: strOrEmpty(data.(map[string]interface{})["target"].(map[string]interface{})["body"]),
				Uri:  strOrEmpty(data.(map[string]interface{})["target"].(map[string]interface{})["uri"]),
				Transform: transform,
			},
			Mapping: func() map[string][]string {
				tmp := map[string][]string{}
				for key, value := range data.(map[string]interface{})["mapping"].(map[string]interface{}) {
					if str, ok := value.(string); ok {
						tmp[key] = []string{str}
					} else if arr, ok := value.([]interface{}); ok {
						tmp[key] = make([]string, 0)
						for _, v := range arr {
							tmp[key] = append(tmp[key], v.(string))
						}
					} else {
						log.Fatal(fmt.Errorf("unsupported mapping %s[%q] = %q", id, key, value))
					}
				}
				return tmp
			}(),
			Caching : cache,
		}

		if len(m.Mapping) == 0 {
			return nil, fmt.Errorf("ignored mapping as it contained no mappings: %s", id)
		}

		if compiled, err := m.Compile(); err != nil {
			return nil, err
		} else {
			log.Printf("loaded mapping '%v'\n", id)
			loadedIds = append(loadedIds, id)
			*list = append(*list, compiled)
		}
	}
	return loadedIds, nil
}

func parseTargetTransform(data interface{}) (*TargetTransform, error) {
	if m, exist := data.(map[string]interface{}); exist {
		log.Printf("loading transformation: %v", m)
		t := &TargetTransform{
			Type: m["type"].(string),
			Template: m["template"].(string),
		}

		if value, exists := m["headers"]; exists {
			if headers, ok := value.(map[string]interface{}); ok {
				t.Headers = make(map[string]string)
				for key, raw := range headers {
					if value, ok := raw.(string); ok {
						t.Headers[key] = value
					}
				}
				log.Printf("loaded headers: %v\n", t.Headers)
			}
		}

		if expr, ok := m["regexp"].(string); ok {
			r, err := regexp.Compile(expr)
			if err != nil {
				return nil, err
			}
			log.Printf("compiled regexp: %v", expr)
			t.Regexp = r
		}
		return t, nil
	}
	return nil, nil
}

func Load(config map[string]interface{}) (*Mappings, error) {
	list := make(Mappings, 0)
	_, err := list.Register(config)
	if err != nil {
		return nil, err
	}
	return &list, nil
}

func strOrEmpty(v interface{}) string {
	if v == nil {
		return ""
	} else {
		return v.(string)
	}
}

func boolOrFalse(v interface{}) bool {
	if value, ok := v.(bool); ok {
		return value
	} else {
		return false
	}
}