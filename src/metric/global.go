package metric

import (
	"log"
	"sync"
)

var (
	mu             sync.RWMutex
	defaultEmitter *Emitter
)

func Setup(opts ...EmitterOpt) {
	mu.Lock()
	defer mu.Unlock()
	var err error
	defaultEmitter, err = New(opts...)
	if err != nil {
		log.Panic(err)
	}
}

func IncCounter(name string, options ...IncrementOpt) {
	mu.RLock()
	defer mu.RUnlock()
	defaultEmitter.IncCounter(name, options...)
}

func PulseCounter(name string, options ...PulseOpt) func(delta uint64) {
	mu.RLock()
	defer mu.RUnlock()
	return defaultEmitter.PulseCounter(name, options...)
}