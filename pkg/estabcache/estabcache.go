package estabcache

import (
	"log"
	"sync"
	"time"
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

func (e *EstablishedCache) PrintConnections() {
	e.rw.RLock()
	log.Print(e.m)
	e.rw.RUnlock()
}
