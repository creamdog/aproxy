package main

import(
	"io/ioutil"
	"encoding/json"
)

const (
	configFile = "config.json"
)

type Config struct {
	Listeners []map[string]interface{} `json:"listeners"`
	Mappings map[string]interface{} `json:"mappings"`
}

func LoadConfig(filename string) (*Config, error) {
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