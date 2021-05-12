package main

import "sync"

type SpeedCache struct {
	sync.Mutex
	cache map[string]string
}

func makeSpeedMap() *SpeedCache {
	sc := &SpeedCache{}
	mp := make(map[string]string)
	mp["daa-128"] = "128"
	mp["daa-128-zero"] = "128"
	mp["daa-128-kachok"] = "128"
	mp["daa-512"] = "782"
	mp["daa-2048-game"] = "2148"
	mp["daa-1024"] = "1124"
	mp["daa-256"] = "286"
	mp["daa-128-12"] = "128"
	mp["daa-128-12"] = "128"
	mp["daa-256-20"] = "286"
	mp["daa-512-30"] = "786"
	mp["daa-1024-40"] = "1124"
	mp["daa-2048-60"] = "2148"
	mp["daa-4096"] = "4196"
	mp["daa-6144"] = "6244"
	mp["daa-8192"] = "8292"
	mp["daa-10280"] = "10240"
	mp["daa-10240"] = "10240"
	mp["daa-512-game"] = "782"
	mp["daa-256-game"] = "286"
	mp["daa-1024-game"] = "1124"
	mp["daa-102400"] = "103240"
	mp["daa-50000"] = "53240"
	mp["daa-3072"] = "3072"
	mp["daa-5120"] = "5120"
	mp["daa-20480"] = "20480"
	mp["daa-20000"] = "20480"
	mp["daa-30720"] = "30720"
	mp["daa-30000"] = "30720"
	mp["daa-40000"] = "40960"
	mp["daa-50000"] = "63240"
	mp["daa-36000"] = "36000"
	mp["daa-70000"] = "73240"
	sc.cache = mp
	return sc
}
func (sc *SpeedCache) GetSpeed(key string) (value string) {
	sc.Lock()
	defer sc.Unlock()
	value = sc.getSpeed(key)
	if value == "" {
		value = "128"
	}
	return
}
func (sc *SpeedCache) getSpeed(key string) (value string) {
	value = sc.cache[key]
	return
}
