package config

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var (
	globalConfig     map[string]interface{}
	configMutex      sync.RWMutex
	configInitialized bool
)

type Config struct {
	data map[string]interface{}
}

func Init(configPath string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	if configInitialized {
		return fmt.Errorf("config already initialized")
	}

	data, err := parseYAMLFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	globalConfig = data
	configInitialized = true

	return nil
}

func Get(key string) (interface{}, error) {
	configMutex.RLock()
	defer configMutex.RUnlock()

	if !configInitialized {
		return nil, fmt.Errorf("config not initialized")
	}

	return getValue(globalConfig, key)
}

func GetString(key string) (string, error) {
	value, err := Get(key)
	if err != nil {
		return "", err
	}

	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("config value for key '%s' is not a string", key)
	}

	return str, nil
}

func GetInt(key string) (int, error) {
	value, err := Get(key)
	if err != nil {
		return 0, err
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("config value for key '%s' is not an integer", key)
	}
}

func GetBool(key string) (bool, error) {
	value, err := Get(key)
	if err != nil {
		return false, err
	}

	boolean, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("config value for key '%s' is not a boolean", key)
	}

	return boolean, nil
}

func getValue(data map[string]interface{}, key string) (interface{}, error) {
	keys := strings.Split(key, ".")
	current := data

	for i, k := range keys {
		value, exists := current[k]
		if !exists {
			return nil, fmt.Errorf("config key '%s' not found", key)
		}

		if i == len(keys)-1 {
			return value, nil
		}

		nextMap, ok := value.(map[string]interface{})
		if !ok {
			v := reflect.ValueOf(value)
			if v.Kind() == reflect.Map {
				nextMap = make(map[string]interface{})
				for _, mapKey := range v.MapKeys() {
					nextMap[fmt.Sprintf("%v", mapKey.Interface())] = v.MapIndex(mapKey).Interface()
				}
			} else {
				return nil, fmt.Errorf("config key '%s' is not a map at level '%s'", key, strings.Join(keys[:i+1], "."))
			}
		}
		current = nextMap
	}

	return nil, fmt.Errorf("unexpected error getting config value for key '%s'", key)
}