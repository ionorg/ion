package auth

import "github.com/golang-jwt/jwt/v4"

// claims custom claims type for jwt
type Claims struct {
	UID      string   `json:"uid"`
	SID      string   `json:"sid"`
	Publish  bool     `json:"publish"`
	Subcribe bool     `json:"subscribe"`
	Services []string `json:"services"`
	jwt.StandardClaims
}
