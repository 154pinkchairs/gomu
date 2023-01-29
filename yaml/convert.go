// convert.go converts anko key-value pairs to yaml key-value pairs.
package yaml

import (
	"fmt"
	"reflect"
	"strings"

	_ "gopkg.in/yaml.v3"
)

func Convert(ankoMap map[string]interface{}) (yamlMap map[string]interface{}, err error) {
	yamlMap = make(map[string]interface{})
	for k, v := range ankoMap {
		if err = convert(yamlMap, k, v); err != nil {
			return
		}
	}
	return
}

func convert(yamlMap map[string]interface{}, key string, value interface{}) (err error) {
	if strings.Contains(key, ".") {
		keys := strings.Split(key, ".")
		if len(keys) > 2 {
			err = fmt.Errorf("key %s is too deep", key)
			return
		}
		if _, ok := yamlMap[keys[0]]; !ok {
			yamlMap[keys[0]] = make(map[string]interface{})
		}
		if err = convert(yamlMap[keys[0]].(map[string]interface{}), keys[1], value); err != nil {
			return
		}
	} else {
		switch value.(type) {
		case string:
			yamlMap[key] = value.(string)
		case int:
			yamlMap[key] = value.(int)
		case bool:
			yamlMap[key] = value.(bool)
		default:
			err = fmt.Errorf("unsupported type %s", reflect.TypeOf(value))
		}
	}
	return
}
