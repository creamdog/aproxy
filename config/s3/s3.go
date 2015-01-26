package s3

import (
	"encoding/json"
	"github.com/creamdog/aproxy/mappings"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"log"
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

func (w *worker) poll() {

	for {
		more := true
		marker := ""
		seen := make([]string, 0)
		for more {
			more = false
			resp, err := w.bucket.List(w.enabledPrefix, "/", marker, 1000)
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
				log.Printf("de-registering mapping '%s'", key)
				w.mapping.DeRegister(key)
				delete(w.seen, key)
			}
		}

		time.Sleep(5 * time.Second)
	}
}
