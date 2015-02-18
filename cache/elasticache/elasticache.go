package elasticache

import (
	"../memcached"
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/elasticache"
)

func Init(config map[string]interface{}) (*memcached.MemcachedClient, error) {
	auth := &aws.Auth{AccessKey: config["access_key"].(string), SecretKey: config["secret_key"].(string)}
	client := elasticache.New(*auth, aws.GetRegion(config["region"].(string)))
	if info, err := client.DescribeCacheCluster(config["cluster"].(string)); err != nil {
		return nil, err
	} else {
		list := []interface{}{}
		for _, node := range info.CacheNodes {
			list = append(list, fmt.Sprintf("%s:%d", node.Endpoint.Host, node.Endpoint.Port))
		}
		config["hosts"] = list
		return memcached.Init(config)
	}
}
