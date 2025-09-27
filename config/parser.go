package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

func parseYAMLFile(configPath string) (map[string]interface{}, error) {
	ext := filepath.Ext(configPath)
	if ext != ".yaml" && ext != ".yml" {
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}

	return config, nil
}