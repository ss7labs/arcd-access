package main

import "sync"

type SessCache struct {
	sync.Mutex // ← этот мьютекс защищает кэш ниже
	cache      map[byte][16]byte
}

func New() *SessCache {
	return &SessCache{
		cache: make(map[byte][16]byte),
	}
}

func (sc *SessCache) set(key byte, value [16]byte) {
	sc.cache[key] = value
}
func (sc *SessCache) get(key byte) (value [16]byte) {
	if len(sc.cache) > 0 {
		value = sc.cache[key]
	}
	return
}
func (sc *SessCache) delete(key byte) {
	_, ok := sc.cache[key]
	if ok {
		delete(sc.cache, key)
	}
}
func (sc *SessCache) Set(key byte, value [16]byte) {
	sc.Lock()
	defer sc.Unlock()
	sc.set(key, value)
}
func (sc *SessCache) Get(key byte) (value [16]byte) {
	sc.Lock()
	defer sc.Unlock()
	value = sc.get(key)
	sc.delete(key)
	return
}
