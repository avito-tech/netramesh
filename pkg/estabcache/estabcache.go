package estabcache

import (
	"sync"
	"time"

	"github.com/Lookyan/netramesh/pkg/log"
)

type EstablishedCache struct {
	rw sync.RWMutex
	m  map[string]string
}

func NewEstablishedCache() *EstablishedCache {
	return &EstablishedCache{
		rw: sync.RWMutex{},
		m:  make(map[string]string),
	}
}

func (e *EstablishedCache) Add(addr string) {
	e.rw.Lock()
	e.m[addr] = time.Now().String()
	e.rw.Unlock()
}

func (e *EstablishedCache) Remove(addr string) {
	e.rw.Lock()
	delete(e.m, addr)
	e.rw.Unlock()
}

func (e *EstablishedCache) PrintConnections(logger *log.Logger) {
	e.rw.RLock()
	logger.Info(e.m)
	e.rw.RUnlock()
}
