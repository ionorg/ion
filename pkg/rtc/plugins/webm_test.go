package plugins

import (
	"testing"
)

func TestWebMSaver(t *testing.T) {
	saver := NewWebmSaver("id")
	saver.Stop()
}
