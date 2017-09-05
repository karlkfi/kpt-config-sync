package adapter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

func Load(filename string) ([]PolicyNode, error) {
	config, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Error reading file: %s", err)
	}
	var nodes []PolicyNode
	err = json.Unmarshal(config, &nodes)
	if err != nil {
		return nil, fmt.Errorf("Error parsing file: %s", err)
	}
	return nodes, nil
}
