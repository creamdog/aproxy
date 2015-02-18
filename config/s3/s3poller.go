package s3

import (
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"log"
	"time"
)

type S3Poller struct {
	Auth           *aws.Auth
	S3Client       *s3.S3
	Bucket         *s3.Bucket
	EnabledPrefix  string
	Seen           map[string]string
	HandleAddition func([]byte, s3.Key) error
	HandleRemoval  func(string) error
}

func (w *S3Poller) poll() {

	for {
		more := true
		marker := ""
		newSeen := make([]string, 0)
		for more {
			more = false
			resp, err := w.Bucket.List(w.EnabledPrefix, "", marker, 1000)
			if err != nil {
				log.Printf("%q", err)
				continue
			}
			for _, content := range resp.Contents {
				newSeen = append(newSeen, content.Key)
				if m, exist := w.Seen[content.Key]; exist && m == content.LastModified {
					continue
				}
				w.Seen[content.Key] = content.LastModified
				log.Printf("fetching mapping configuration %s", content.Key)
				data, err := w.Bucket.Get(content.Key)
				if err != nil {
					log.Printf("%q", err)
					continue
				}

				err = w.HandleAddition(data, content)
				if err != nil {
					log.Printf("%q", err)
					continue
				}
			}
			more = resp.IsTruncated
			marker = resp.Marker
		}

		for key, _ := range w.Seen {
			found := false
			for _, key2 := range newSeen {
				if key == key2 {
					found = true
					break
				}
			}
			if !found {
				log.Printf("de-registering mappings %q", w.Seen[key])
				err := w.HandleRemoval(key)
				if err != nil {
					log.Printf("%q", err)
					continue
				}
				delete(w.Seen, key)
			}
		}

		time.Sleep(5 * time.Second)
	}
}
