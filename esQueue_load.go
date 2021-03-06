// esQueue

// +build arm arm64 mips64 mips64le ppc64 ppc64le

package queue

import (
	"runtime"
	"sync/atomic"
)

func (q *EsQueue) Quantity() uint32 {
	var putPos, getPos uint32
	var quantity uint32
	getPos = atomic.LoadUint32(&q.getPos)
	putPos = atomic.LoadUint32(&q.putPos)

	if putPos >= getPos {
		quantity = putPos - getPos
	} else {
		quantity = q.capMod - getPos + putPos
	}

	return quantity
}

// put queue functions
func (q *EsQueue) Put(val interface{}) (ok bool, quantity uint32) {
	var putPos, putPosNew, getPos, posCnt uint32
	var cache *esCache
	capMod := q.capMod
	for {
		getPos = atomic.LoadUint32(&q.getPos)
		putPos = atomic.LoadUint32(&q.putPos)

		if putPos >= getPos {
			posCnt = putPos - getPos
		} else {
			posCnt = capMod - getPos + putPos
		}

		if posCnt >= capMod {
			runtime.Gosched()
			return false, posCnt
		}

		putPosNew = putPos + 1
		if atomic.CompareAndSwapUint32(&q.putPos, putPos, putPosNew) {
			break
		} else {
			runtime.Gosched()
		}
	}

	cache = &q.cache[putPosNew&capMod]

	for {
		if !cache.mark {
			cache.value = val
			cache.mark = true
			return true, posCnt + 1
		} else {
			runtime.Gosched()
		}
	}
}

// get queue functions
func (q *EsQueue) Get() (val interface{}, ok bool, quantity uint32) {
	var putPos, getPos, getPosNew, posCnt uint32
	var cache *esCache
	capMod := q.capMod
	for {
		putPos = atomic.LoadUint32(&q.putPos)
		getPos = atomic.LoadUint32(&q.getPos)

		if putPos >= getPos {
			posCnt = putPos - getPos
		} else {
			posCnt = capMod - getPos + putPos
		}

		if posCnt < 1 {
			runtime.Gosched()
			return nil, false, posCnt
		}

		getPosNew = getPos + 1
		if atomic.CompareAndSwapUint32(&q.getPos, getPos, getPosNew) {
			break
		} else {
			runtime.Gosched()
		}
	}

	cache = &q.cache[getPosNew&capMod]

	for {
		if cache.mark {
			val = cache.value
			cache.value = nil
			cache.mark = false
			return val, true, posCnt - 1
		} else {
			runtime.Gosched()
		}
	}
}
