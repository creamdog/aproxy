package config

type Configuration struct {
	Mappings map[string]interface{} `json:"mappings"`
}

type ConfigurationManager interface {
	Get() []*Configuration
	SaveMapping(mapping map[string]interface{})
	DeleteMapping()
}
