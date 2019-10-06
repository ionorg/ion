package util

import (
	"fmt"
	"testing"
)

func TestMarshal(t *testing.T) {
	m := make(map[string]interface{})
	m["abc"] = "123"
	str := Marshal(m)
	fmt.Println(str)
}

func TestUnmarshal(t *testing.T) {
	str := "{\"abc\": 123}"
	m := Unmarshal(str)
	fmt.Println(m)
}
