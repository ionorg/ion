package sctp

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"
)

type paramStateCookie struct {
	paramHeader
	cookie []byte
}

func newRandomStateCookie() *paramStateCookie {
	rs := rand.NewSource(time.Now().UnixNano())
	r := rand.New(rs)
	randCookie := make([]byte, 32)
	i := 0
	for i < 4 {
		binary.BigEndian.PutUint64(randCookie[i*4:], r.Uint64())
		i++
	}

	s := &paramStateCookie{
		cookie: randCookie,
	}

	return s
}

func (s *paramStateCookie) marshal() ([]byte, error) {
	s.typ = stateCookie
	s.raw = s.cookie
	return s.paramHeader.marshal()
}

func (s *paramStateCookie) unmarshal(raw []byte) (param, error) {
	err := s.paramHeader.unmarshal(raw)
	if err != nil {
		return nil, err
	}
	s.cookie = s.raw
	return s, nil
}

// String makes paramStateCookie printable
func (s *paramStateCookie) String() string {
	return fmt.Sprintf("%s: %s", s.paramHeader, s.cookie)
}
