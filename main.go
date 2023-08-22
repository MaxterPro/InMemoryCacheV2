package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

const (
	INFINITY = -1
	DEFAULT  = 0
)

func main() {
	t := time.Now()
	fmt.Println("Hello World")
	cache := New(10*time.Hour, 20*time.Minute)
	fmt.Println(cache.defaultExpiryDuration)
	fmt.Println(cache.storage)

	fmt.Println(cache.storage)
	value, found := cache.Get("foo")
	if found {
		fmt.Println("Value is ", value)
	}
	fmt.Println("Time ", time.Since(t))
}

type Data struct {
	Value    interface{}
	ExpireAt int64
}

type Cleaner struct {
	Interval time.Duration
	stop     chan bool
}

type cache struct {
	defaultExpiryDuration time.Duration
	storage               map[string]Data
	locker                sync.RWMutex
	cleaner               *Cleaner
	onRemoval             func(string, interface{})
}

type Cache struct {
	*cache
}

func New(defaultExpiryDuration time.Duration, cleanUpInterval time.Duration) *Cache {
	if defaultExpiryDuration == 0 {
		defaultExpiryDuration = INFINITY
	}

	cache := &cache{
		defaultExpiryDuration: defaultExpiryDuration,
		storage:               make(map[string]Data),
	}

	Cache := &Cache{cache}

	if cleanUpInterval > 0 {
		clean(cleanUpInterval, cache)
		runtime.SetFinalizer(Cache, stopCleaning)
	}
	return Cache
}

func clean(cleanUpInterval time.Duration, cache *cache) {
	cleaner := &Cleaner{
		Interval: cleanUpInterval,
		stop:     make(chan bool),
	}

	cache.cleaner = cleaner
	go cleaner.Cleaning(cache)

}

func (c *Cleaner) Cleaning(cache *cache) {
	ticker := time.NewTicker(c.Interval)

	for {
		select {
		case <-ticker.C:
			cache.purge()
		case <-c.stop:
			ticker.Stop()

		}
	}
}

func stopCleaning(cache *Cache) {
	cache.cleaner.stop <- true
}

func (cache *cache) purge() {
	now := time.Now().UnixNano()
	for key, data := range cache.storage {
		if data.ExpireAt < now {
			delete(cache.storage, key)
		}
	}
}

func (c *cache) Set(key string, value interface{}, LifeTime time.Duration) {
	if LifeTime == DEFAULT {
		LifeTime = c.defaultExpiryDuration
	}
	var expireAt int64

	if LifeTime > 0 {
		expireAt = time.Now().Add(LifeTime).UnixNano()
	}
	c.locker.Lock()
	defer c.locker.Unlock()
	c.storage[key] = Data{
		Value:    value,
		ExpireAt: expireAt,
	}
}
func (c *cache) Get(key string) (interface{}, bool) {
	c.locker.RLock()
	defer c.locker.RUnlock()
	data, found := c.storage[key]
	if !found {
		return nil, false
	}

	if data.ExpireAt < time.Now().UnixNano() {
		return nil, false
	}

	return data.Value, true
}
