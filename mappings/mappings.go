package mappings

import(
	"text/template"
	"regexp"
	"log"
	"bytes"
	"fmt"
)

type Mapping struct {
	Id string
	Target *TargetMapping
	Mapping map[string][]string
}
type TargetMapping struct {
	Headers map[string]string
	Verb string
	Body string
	Uri string	
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
	compiledMappings := map[string][]*regexp.Regexp{}
	for key, values := range q.Mapping {
		for _, value := range values {
			compiledRegexp, err := regexp.Compile(value)
			if err != nil {
				return nil, err
			}
			if _,exists := compiledMappings[key]; !exists {
				compiledMappings[key] = make([]*regexp.Regexp, 0)
			}
			compiledMappings[key] = append(compiledMappings[key], compiledRegexp)
			log.Printf("compiled mapping %v[%v](%v)\n", q.Id, key, value)
		}
	}

	return &CompiledMapping{
		Mapping: q,
		CompiledBody : body,
		CompiledUrl : url,
		CompiledMapping : compiledMappings,
	}, nil
}

type CompiledMapping struct {
	Mapping *Mapping
	CompiledBody *template.Template
	CompiledUrl *template.Template
	CompiledMapping map[string][]*regexp.Regexp
}

type RequestMapping struct {
	Id string
	Body string
	Headers map[string]string
	Verb string
	Uri string
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
	return &RequestMapping {
		Id : cm.Mapping.Id,
		Body : body,
		Headers : cm.Mapping.Target.Headers,
		Verb : cm.Mapping.Target.Verb,
		Uri : uri,
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
			for k, v := range flatten(keypath + key, m) {
				tmp[k] = v
			}
		} else if values, ok := value.([]string); ok {
			tmp[keypath + key] = values
		} else {
			tmp[keypath + key] = value.(string)
		}
	}
	return tmp
}

func (m Mappings) GetMatch(complexData map[string]interface{}) (*RequestMapping, error) {
	data := flatten("", complexData)
	for _,cm := range m {
		isMatch := true
		for key, regexpList := range cm.CompiledMapping {
			log.Printf("checking %q, key: %v, data: %q", m, key, data)
			if value, exists := data[key]; !exists {
				isMatch = false
				break
			} else {
				for _, regexp := range regexpList {
					if values,ok := value.([]string); ok {
						for _,value := range values {
							log.Printf("checking VALUE %q, key: %v, value: %v", m, key, value)
							if regexp.Match([]byte(value)) {
								break
							}							
							isMatch = false
						}
					} else if value, ok := value.(string); ok {
						log.Printf("checking VALUE %q, key: %v, value: %v", m, key, value)
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

func Load(config map[string]interface{}) (Mappings, error) {
	list := make([]*CompiledMapping, 0)
	for id, data := range config {
		m := &Mapping{
			Id : id,
			Target : &TargetMapping{
				Headers : func()map[string]string {
						tmp :=  data.(map[string]interface{})["target"].(map[string]interface{})["headers"].(map[string]interface{})
						list := map[string]string{}
						for key, v := range tmp {
							list[key] = v.(string)
						}
						return list
					}(),
				Verb : strOrEmpty(data.(map[string]interface{})["target"].(map[string]interface{})["verb"]),
				Body : strOrEmpty(data.(map[string]interface{})["target"].(map[string]interface{})["body"]),
				Uri : strOrEmpty(data.(map[string]interface{})["target"].(map[string]interface{})["uri"]),
			},
			Mapping : func() map[string][]string {
					tmp := map[string][]string{}
					for key, value := range data.(map[string]interface{})["mapping"].(map[string]interface{}) {
						if str, ok := value.(string); ok {
							tmp[key] = []string{str}
						} else if arr, ok := value.([]interface{}); ok {
							tmp[key] = make([]string, 0)
							for _,v := range arr {
								tmp[key] = append(tmp[key], v.(string))
							} 
						} else {
							log.Fatal(fmt.Errorf("unsupported mapping %s[%q] = %q", id, key, value))
						}
					}
					return tmp
				}(),
		}
		if compiled, err := m.Compile(); err != nil {
			return nil, err
		} else {
			log.Printf("loaded mapping '%v'\n", id)
			list = append(list, compiled)
		}
	}
	return list, nil
}

func strOrEmpty(v interface{}) string {
	if v == nil {
		return ""
	} else {
		return v.(string)
	}
}