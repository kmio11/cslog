package cslog

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
)

type IDGenerator interface {
	NewID() LogID
}

var logIdGenerator IDGenerator = newRandGen()

// SetLogIdGenerator sets the logIdGenerator which generates logId and parentLogId.
func SetLogIdGenerator(gen IDGenerator) {
	logIdGenerator = gen
}

var _ IDGenerator = (*randGen)(nil)

type randGen struct {
	sync.Mutex
	randSource *rand.Rand
}

func newRandGen() *randGen {
	gen := &randGen{}
	var rngSeed int64
	_ = binary.Read(crand.Reader, binary.LittleEndian, &rngSeed)
	gen.randSource = rand.New(rand.NewSource(rngSeed))
	return gen
}

func (r *randGen) NewID() LogID {
	r.Lock()
	defer r.Unlock()

	id := ByteLogID{}
	_, _ = r.randSource.Read(id[:])

	return id
}
