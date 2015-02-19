package s3

import (
	"encoding/json"
	"github.com/creamdog/aproxy/mappings"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"log"
	"strings"
)

type FilesStatus struct {
	Mappings         *mappings.Mappings
	FileToMappingIds map[string][]string
}

func Start(mapping *mappings.Mappings, config map[string]interface{}) {
	auth := &aws.Auth{AccessKey: config["access_key"].(string), SecretKey: config["secret_key"].(string)}
	s3Client := s3.New(*auth, aws.GetRegion(config["region"].(string)))
	bucket := s3Client.Bucket(config["bucket"].(string))

	filesStatus := FilesStatus{mapping, make(map[string][]string)}

	poller := S3Poller{
		auth,
		s3Client,
		bucket,
		config["enabled_prefix"].(string),
		make(map[string]string),
		filesStatus.makeAdditionHandler(),
		filesStatus.makeRemovalHandler(),
	}

	go poller.poll()
}

func (fs *FilesStatus) makeAdditionHandler() func([]byte, s3.Key) error {
	return func(data []byte, content s3.Key) error {
		dconf := map[string]interface{}{}
		if err := json.Unmarshal(data, &dconf); err != nil {
			log.Printf("%s => %q", content.Key, err)
			return err
		}
		dconf = ConfigMap(dconf).LowerCaseKeys()

		log.Printf(string(data))
		log.Printf("%q", dconf)

		if ids, err := fs.Mappings.Register(dconf["mappings"].(map[string]interface{})); err != nil {
			log.Printf("%q", err)
			return err
		} else {
			fs.FileToMappingIds[content.Key] = ids
			log.Printf("%s => registered ids %q", content.Key, fs.FileToMappingIds[content.Key])
		}

		return nil
	}
}

func (fs *FilesStatus) makeRemovalHandler() func(string) error {
	return func(key string) error {
		fs.Mappings.DeRegister(fs.FileToMappingIds[key])
		return nil
	}
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
