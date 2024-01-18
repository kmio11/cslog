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

var idGenerator IDGenerator = newRandGen()

func SetIDGenerator(gen IDGenerator) {
	idGenerator = gen
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

	id := LogID{}
	_, _ = r.randSource.Read(id[:])

	return id
}
