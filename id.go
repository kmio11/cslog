package cslog

import (
	"encoding/hex"
)

type LogID interface {
	String() string
	IsZero() bool
}

var _ LogID = StringLogID("")

type StringLogID string

func (s StringLogID) String() string {
	return string(s)
}

func (s StringLogID) IsZero() bool {
	return s == ""
}

var _ LogID = ByteLogID{}

type ByteLogID [8]byte

func (id ByteLogID) String() string {
	if id.IsZero() {
		return ""
	}
	return hex.EncodeToString(id[:])
}

// IsZero reports whether id represents the zero instant,
func (id ByteLogID) IsZero() bool {
	return id == ByteLogID{}
}
