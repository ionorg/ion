package signal

import (
	"context"

	"github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthConfig auth config
type AuthConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Key     string `mapstructure:"key"`
	KeyType string `mapstructure:"key_type"`
}

// KeyFunc auth key types
func (a AuthConfig) KeyFunc(t *jwt.Token) (interface{}, error) {
	// nolint: gocritic
	switch a.KeyType {
	//TODO: add more support for keytypes here
	default:
		return []byte(a.Key), nil
	}
}

// claims custom claims type for jwt
type claims struct {
	UID      string   `json:"uid"`
	SID      string   `json:"sid"`
	Services []string `json:"services"`
	jwt.StandardClaims
}

func getClaim(ctx context.Context, ac *AuthConfig) (*claims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "valid JWT token required")
	}

	token, ok := md["authorization"]
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "valid JWT token required")
	}

	jwtToken, err := jwt.ParseWithClaims(token[0], &claims{}, ac.KeyFunc)

	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "%v", err)
	}

	if claims, ok := jwtToken.Claims.(*claims); ok && jwtToken.Valid {
		return claims, nil
	}

	return nil, status.Errorf(codes.Unauthenticated, "valid JWT token required: %v", err)
}
