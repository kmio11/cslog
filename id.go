package cslog

import (
	"encoding/hex"
)

type LogID [8]byte

var Nil = LogID{}

func (id LogID) String() string {
	if id.IsZero() {
		return ""
	}
	return hex.EncodeToString(id[:])
}

// IsZero reports whether id represents the zero instant,
func (id LogID) IsZero() bool {
	return id == Nil
}
