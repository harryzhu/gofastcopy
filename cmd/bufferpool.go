package cmd

import (
	"sync"
)

type BufferPool struct {
	emptyBytes []byte
	pool       sync.Pool
}

func NewBufferPool(size, cap int) *BufferPool {
	bp := &BufferPool{}
	bp.emptyBytes = make([]byte, size)
	bp.pool.New = func() any {
		b := make([]byte, size)
		return b
	}
	return bp
}

func (bp *BufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

func (bp *BufferPool) Put(b []byte) {
	b = bp.emptyBytes
	bp.pool.Put(b)
}
