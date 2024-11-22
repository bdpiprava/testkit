package search

import (
	"encoding/json"
	"fmt"
)

const createIndexBodyTemplate = `{
	"settings": {
		"index": {
			"number_of_shards": %d,
			"number_of_replicas": %d
		}
	},
	"mappings": {
		"dynamic": %t,
		"properties": %s
	}
}`

type CreateIndexSettings struct {
	NumberOfShards          int
	NumberOfReplicas        int
	Dynamic                 bool
	MappingProperties       map[string]any
	MappingPropertiesString string
}

func (c *CreateIndexSettings) GetBody() (string, error) {
	props := c.MappingPropertiesString
	if props == "" && len(c.MappingProperties) > 0 {
		propBytes, err := json.Marshal(c.MappingProperties)
		if err != nil {
			return "", err
		}
		props = string(propBytes)
	}

	if c.NumberOfShards == 0 {
		c.NumberOfShards = 1
	}

	if c.NumberOfReplicas == 0 {
		c.NumberOfReplicas = 1
	}

	return fmt.Sprintf(
		createIndexBodyTemplate,
		c.NumberOfShards,
		c.NumberOfReplicas,
		c.Dynamic, props,
	), nil
}
