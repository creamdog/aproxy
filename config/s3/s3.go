package s3

import (
	"../../mappings"
	"encoding/json"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"log"
	"strings"
	"time"
)

type worker struct {
	auth          *aws.Auth
	s3Client      *s3.S3
	bucket        *s3.Bucket
	mapping       *mappings.Mappings
	enabledPrefix string
	config        map[string]interface{}
	seen          map[string]*ConfigStatus
}

type ConfigStatus struct {
	ids      []string
	modified string
}

func Start(mapping *mappings.Mappings, config map[string]interface{}) {
	auth := &aws.Auth{AccessKey: config["access_key"].(string), SecretKey: config["secret_key"].(string)}
	s3Client := s3.New(*auth, aws.GetRegion(config["region"].(string)))
	bucket := s3Client.Bucket(config["bucket"].(string))
	w := &worker{auth, s3Client, bucket, mapping, config["enabled_prefix"].(string), config, make(map[string]*ConfigStatus)}
	go w.poll()
}

type ConfigMap map[string]interface{}

func (m ConfigMap) LowerCaseKeys() map[string]interface{} {
	tmp := map[string]interface{}{}
	for key, value := range m {
		key = strings.ToLower(key)
		if subMap, isMap := value.(map[string]interface{}); isMap && key != "headers" && key != "mapping" {
			tmp[key] = ConfigMap(subMap).LowerCaseKeys()
		} else {
			tmp[key] = value
		}
	}
	return tmp
}

func (w *worker) poll() {

	for {
		more := true
		marker := ""
		seen := make([]string, 0)
		for more {
			more = false
			resp, err := w.bucket.List(w.enabledPrefix, "", marker, 1000)
			if err != nil {
				log.Printf("%q", err)
				continue
			}
			for _, content := range resp.Contents {
				seen = append(seen, content.Key)
				if m, exist := w.seen[content.Key]; exist && m.modified == content.LastModified {
					continue
				}
				w.seen[content.Key] = &ConfigStatus{make([]string, 0), content.LastModified}
				log.Printf("fetching mapping configuration %s", content.Key)
				data, err := w.bucket.Get(content.Key)
				if err != nil {
					log.Printf("%q", err)
					continue
				}
				dconf := map[string]interface{}{}
				if err = json.Unmarshal(data, &dconf); err != nil {
					log.Printf("%s => %q", content.Key, err)
					continue
				}
				dconf = ConfigMap(dconf).LowerCaseKeys()

				log.Printf(string(data))
				log.Printf("%q", dconf)

				if ids, err := w.mapping.Register(dconf["mappings"].(map[string]interface{})); err != nil {
					log.Printf("%q", err)
					continue
				} else {
					w.seen[content.Key].ids = ids
					log.Printf("%s => registered ids %q", content.Key, w.seen[content.Key].ids)
				}
			}
			more = resp.IsTruncated
			marker = resp.Marker
		}

		for key, _ := range w.seen {
			found := false
			for _, key2 := range seen {
				if key == key2 {
					found = true
					break
				}
			}
			if !found {
				log.Printf("de-registering mappings %q", w.seen[key].ids)
				w.mapping.DeRegister(w.seen[key].ids)
				delete(w.seen, key)
			}
		}

		time.Sleep(5 * time.Second)
	}
}
