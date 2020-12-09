package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

// key under [signal.auth_connection] of configs/biz.toml
const key = "1q2dGu5pzikcrECJgW3ADfXX3EsmoD99SYvSVCpDsJrAqxou5tUNbHPvkEFI4bTS"

type result struct {
	Token string `json:"token"`
}

type claims struct {
	UID string `json:"uid"`
	SID string `json:"sid"`
	jwt.StandardClaims
}

func main() {
	addr := flag.String("addr", ":8080", "server listen address")
	dir := flag.String("dir", "", "Static file directory")
	certFile := flag.String("cert", "", "Certificate file")
	keyFile := flag.String("key", "", "Private key file")
	flag.Parse()

	if len(*addr) <= 0 {
		log.Fatal("No listen address")
	}

	// http://localhost:8080/generate?uid=tony&sid=room1
	http.HandleFunc("/generate", sign)

	// http://localhost:8080/validate?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiJ0b255IiwicmlkIjoicm9vbTEifQ.mopgibW3OYONYwzlo-YvkDIkNoYJc3OBQRsqQHZMnD8
	http.HandleFunc("/validate", validate)

	if len(*dir) > 0 {
		http.Handle("/", http.FileServer(http.Dir(*dir)))
	}

	var url string
	useTLS := len(*certFile) > 0 && len(*keyFile) > 0
	if useTLS {
		url = "https://"
	} else {
		url = "http://"
	}
	if (*addr)[0] == ':' {
		url = url + "localhost" + *addr
	} else {
		url = url + *addr
	}
	fmt.Printf("web server: %s\n", url)

	var err error
	if len(*certFile) > 0 && len(*keyFile) > 0 {
		err = http.ListenAndServeTLS(*addr, *certFile, *keyFile, nil)
	} else {
		err = http.ListenAndServe(*addr, nil)
	}
	if err != nil {
		log.Fatal(err)
	}
}

// building and signing a token
func sign(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()["uid"]
	if values == nil || len(values) < 1 {
		log.Printf("invalid uid, %v", values)
		http.Error(w, "invalid uid", http.StatusForbidden)
		return
	}
	uid := values[0]

	values = r.URL.Query()["sid"]
	if values == nil || len(values) < 1 {
		log.Printf("invalid sid, %v", values)
		http.Error(w, "invalid sid", http.StatusForbidden)
		return
	}
	sid := values[0]

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UID: uid,
		SID: sid,
	})
	tokenString, err := token.SignedString([]byte(key))
	if err != nil {
		log.Printf("sign error: %v", err)
		http.Error(w, "sign error", http.StatusInternalServerError)
	}

	data, err := json.Marshal(result{
		Token: tokenString,
	})
	if err != nil {
		log.Printf("json marshal error: %v", err)
		http.Error(w, "json marshal error", http.StatusInternalServerError)
	}
	log.Printf(string(data))
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// parsing and validating a token
func validate(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()["token"]
	if values == nil || len(values) < 1 {
		log.Printf("invalid token, %v", values)
		http.Error(w, "invalid token", http.StatusForbidden)
		return
	}
	tokenString := values[0]

	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})
	if err != nil {
		log.Printf("parse token error: %v", err)
		http.Error(w, "parse token error", http.StatusForbidden)
		return
	}
	c := token.Claims.(*claims)

	data, err := json.Marshal(c)
	if err != nil {
		log.Printf("json marshal error: %v", err)
		http.Error(w, "json marshal error", http.StatusInternalServerError)
	}
	log.Printf(string(data))
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
