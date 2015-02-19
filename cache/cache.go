package cache

import(
	"github.com/creamdog/aproxy/cache/memcached"
	"github.com/creamdog/aproxy/cache/elasticache"
	"fmt"
)

type CacheClient interface {
	Get(key string, v interface{}) (bool, error)
	Set(key string, expiration int, v interface{}) error
	Delete(key string) error
	FlushAll() error
}


func Get(config map[string]interface{}) (CacheClient, error) {
	if t, exists := config["type"].(string); exists {
		switch t {
			case "memcached" :
				return memcached.Init(config)
			case "elasticache" :
				return elasticache.Init(config)
			default:
				return nil, fmt.Errorf("unsupported cache type: %s", t)
		}	
	} 
	return &NoopClient{}, nil
}

type NoopClient struct {

}

func (np *NoopClient) Get(key string, v interface{}) (bool, error) {
	return false, nil
}

func (np *NoopClient) Delete(key string) error {
	return nil
}

func (np *NoopClient) Set(key string, expiration int, v interface{}) error {
	return nil
}

func (np *NoopClient) FlushAll() error {
	return nil
}