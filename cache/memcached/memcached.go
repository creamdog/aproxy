package memcached

import(
	"github.com/bradfitz/gomemcache/memcache"
	"log"
	"encoding/json"
	"fmt"
	"crypto/sha256"
	"encoding/hex"
)

type MemcachedClient struct {
	client *memcache.Client	
}

func Init(config map[string]interface{}) (*MemcachedClient, error) {
	log.Printf("initializing memcached client: %v", config)

	hosts := []string{}
	if values, exists := config["hosts"].([]interface{}); !exists {
		return nil, fmt.Errorf("no hosts specified for memcached client")
	} else {
		for _,v := range values {
			hosts = append(hosts, v.(string))
		}
	}

	//"172.28.128.3:11211", "10.0.0.2:11211", "10.0.0.3:11212"
	mc := memcache.New(hosts...)
	var c = &MemcachedClient{
		client : mc,
	}
	if _, err := c.Get("test", nil); err != nil {
		return nil, err
	}
	return c, nil

    //mc.Set(&memcache.Item{Key: "foo", Value: []byte("my value")})
}

func Sha256Key(key string) string {
	hash := sha256.New()
	hash.Write([]byte(key))
	md := hash.Sum(nil)
	return hex.EncodeToString(md)
}

func (mc *MemcachedClient) Get(key string, v interface{}) (bool, error) {
	//log.Printf("kjhdskfjhsdfkj>>>>>>>>>>>>>>>>>>>>> %d ==== %s", len(Sha256Key(key)), Sha256Key(key))
	if item, err := mc.client.Get(Sha256Key(key)); err == nil {
		if v != nil {
			if err := json.Unmarshal(item.Value, v); err != nil {
				return false, err
			}
		}
		return true, nil
	} else if err == memcache.ErrCacheMiss {
		return false, nil
	} else {
		return false, err
	}
}

func (mc *MemcachedClient) Set(key string, expiration int, v interface{}) error {
	if bytes, err := json.Marshal(v); err != nil {
		return err
	} else {
		//log.Printf("set cache: %s", string(bytes))
		mc.client.Set(&memcache.Item{Key: Sha256Key(key), Value: bytes, Expiration: int32(expiration)})
	}
	return nil
}

func (mc *MemcachedClient) Delete(key string) error {
	return mc.client.Delete(Sha256Key(key))
}

func (mc *MemcachedClient) FlushAll() error {
	return nil
}