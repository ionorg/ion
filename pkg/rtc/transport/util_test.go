package transport

import (
	"testing"
)

func TestKvOK(t *testing.T) {
	m := make(map[string]interface{})
	m["abc"] = "true"
	if !KvOK(m, "abc", "true") {
		t.Fatal("flag is not true!")
	}
	m["abc"] = 1
	if KvOK(m, "abc", "true") {
		t.Fatal("flag is not true!")
	}
}

func TestValUpper(t *testing.T) {
	m := make(map[string]interface{})
	m["abc"] = "true"
	if ValUpper(m, "abc") != "TRUE" {
		t.Fatal("val is not true!")
	}
	m["abc"] = 1
	if ValUpper(m, "abc") != "" {
		t.Fatal("ValUpper error!")
	}
}
