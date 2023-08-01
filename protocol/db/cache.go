package db

import (
	"github.com/gogf/gf/os/gcache"
)

var cache *gcache.Cache

func init() {
	cache = gcache.New()
}

//获取
func GetCaChe() *gcache.Cache {
	return cache
}
