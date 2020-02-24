package transport

import (
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

// ValUpper get upper string by key
func ValUpper(m map[string]interface{}, k string) string {
	str := ""
	val, ok := m[k]
	if ok {
		str, ok = val.(string)
		if ok {
			return strings.ToUpper(str)
		}
	}
	return ""
}
