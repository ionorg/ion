package util

import (
	"math/rand"
	"runtime"
	"runtime/debug"
	"strings"
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

func GetRedisRoomKey(sid string) string {
	return "/ion/room/" + sid
}

func GetRedisPeerKey(sid, uid string) string {
	return "/ion/room/" + sid + "/" + uid
}

func GetRedisPeersPrefixKey(sid string) string {
	return "/ion/room/" + sid + "/*"
}

func GetArgs(args ...string) (arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10 string) {
	// at least sid uid
	if len(args) < 2 {
		return "", "", "", "", "", "", "", "", "", ""
	}
	// parse args
	for i, arg := range args {
		switch i {
		case 0:
			arg1 = arg
		case 1:
			arg2 = arg
		case 2:
			arg3 = arg
		case 3:
			arg4 = arg
		case 4:
			arg5 = arg
		case 5:
			arg6 = arg
		case 6:
			arg7 = arg
		case 7:
			arg8 = arg
		case 8:
			arg9 = arg
		case 9:
			arg10 = arg
		default:

		}
	}
	return arg1, arg2, arg3, arg4, arg5, arg6, arg7, arg8, arg9, arg10
}

func BoolToString(flag bool) string {
	if !flag {
		return "false"
	}
	return "true"
}

func StringToBool(flag string) bool {
	if strings.ToUpper(flag) == "TRUE" {
		return true
	}
	if flag == "1" {
		return true
	}
	return false
}
