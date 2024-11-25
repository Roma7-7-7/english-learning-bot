package cache

import (
	"sync"
	"time"
)

type InMemory struct {
	storage map[string]string
	lastSet map[string]time.Time

	mx sync.RWMutex
}

func NewInMemory() *InMemory {
	return &InMemory{
		storage: make(map[string]string, 100),
		lastSet: make(map[string]time.Time, 100),

		mx: sync.RWMutex{},
	}
}

func (c *InMemory) Get(key string) (string, bool) {
	c.mx.RLock()
	defer c.mx.RUnlock()

	v, ok := c.storage[key]
	return v, ok
}

func (c *InMemory) Set(key, value string, ttl time.Duration) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.storage[key] = value
	c.lastSet[key] = time.Now()

	go func() {
		time.Sleep(ttl + time.Minute) // add extra minute
		c.mx.Lock()
		defer c.mx.Unlock()
		if _, ok := c.storage[key]; !ok {
			return
		}
		if time.Since(c.lastSet[key]) > ttl {
			delete(c.storage, key)
			delete(c.lastSet, key)
		}
	}()
}
