package config

import (
	"../log"
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Listeners   []map[string]interface{} `json:"listeners"`
	Mappings    map[string]interface{}   `json:"mappings"`
	MappingRepo map[string]interface{}   `json:"mapping"`
	Cache       map[string]interface{}   `json:"cache"`
}

func Load(filename string) (*Config, error) {
	log.Debugf("loading configuration file '%s'", filename)
	if bytes, err := ioutil.ReadFile(filename); err != nil {
		return nil, err
	} else {
		var config Config
		if err = json.Unmarshal(bytes, &config); err != nil {
			return nil, err
		} else {
			return &config, nil
		}
	}
}
