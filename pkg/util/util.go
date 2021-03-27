package util

import (
	"math/rand"
	"runtime"
	"runtime/debug"
	"time"

	log "github.com/pion/ion-log"
)

const (
	DefaultStatCycle   = time.Second * 3
	DefaultGRPCTimeout = 15 * time.Second
)

// RandomString generate a random string
func RandomString(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// Recover print stack
func Recover(flag string) {
	_, _, l, _ := runtime.Caller(1)
	if err := recover(); err != nil {
		log.Errorf("[%s] Recover panic line => %v", flag, l)
		log.Errorf("[%s] Recover err => %v", flag, err)
		debug.PrintStack()
	}
}
