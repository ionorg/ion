package transport

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

// KvOK check flag and value is "true"
func KvOK(m map[string]interface{}, k, v string) bool {
	str := ""
	val, ok := m[k]
	if ok {
		str, ok = val.(string)
		if ok {
			if strings.EqualFold(str, v) {
				return true
			}
		}
	}
	return false
}

// GetUpperString get upper string by key
func GetUpperString(m map[string]interface{}, k string) string {
	val, ok := m[k]
	if ok {
		str, ok := val.(string)
		if ok {
			return strings.ToUpper(str)
		}
	}
	return ""
}

// GetInt get uint64 value by key
func GetInt(m map[string]interface{}, k string) (int, error) {
	val, ok := m[k]
	if ok {
		switch val.(type) {
		case nil:
			return 0, errors.New("value is nil")
		case string:
			i, err := strconv.ParseInt(val.(string), 10, 64)
			if err != nil {
				return 0, err
			}
			return int(i), nil
		case float64:
			return int(val.(float64)), nil
		default:
			return int(reflect.ValueOf(val).Int()), nil
		}
	}
	return 0, errors.New("inavlid key or value type")
}
