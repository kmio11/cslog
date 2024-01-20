package testutil

import (
	"fmt"
	"testing"

	"github.com/kmio11/cslog"
)

var _ (cslog.IDGenerator) = (*CountUpIDGen)(nil)

// CountUpIDGen is IDGenerator to output fixed value.
type CountUpIDGen struct {
	cnt int
}

func (gen *CountUpIDGen) NewID() cslog.LogID {
	id := fmt.Sprintf("%016d", gen.cnt)
	gen.cnt += 1
	return cslog.StringLogID(id)
}

func SetIDGen(t *testing.T) {
	t.Helper()
	cslog.SetLogIdGenerator(&CountUpIDGen{})
}

func ResetIDCount(t *testing.T) {
	t.Helper()
	cslog.SetLogIdGenerator(&CountUpIDGen{})
}
