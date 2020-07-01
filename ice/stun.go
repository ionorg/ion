package ice

import (
	"encoding/binary"
	"fmt"

	"github.com/sssgun/ion/stun"
)

// bin is shorthand for BigEndian.
var bin = binary.BigEndian

func assertInboundUsername(m *stun.Message, expectedUsername string) error {
	var username stun.Username
	if err := username.GetFrom(m); err != nil {
		return err
	}
	if string(username) != expectedUsername {
		return fmt.Errorf("username mismatch expected(%x) actual(%x)", expectedUsername, string(username))
	}

	return nil
}

func assertInboundMessageIntegrity(m *stun.Message, key []byte) error {
	messageIntegrityAttr := stun.MessageIntegrity(key)
	return messageIntegrityAttr.Check(m)
}
