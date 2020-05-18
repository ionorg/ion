package plugins

import (
	"testing"
)

func TestWebMSaver(t *testing.T) {
	saver := NewWebmSaver(WebmSaverConfig{
		ID:   "id",
		Path: "./",
		On:   true,
	})
	saver.Stop()
}
