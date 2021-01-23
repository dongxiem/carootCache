package concurrentcache

import (
	"github.com/Dongxiem/carrotCache/carrotcache/byteview"
	"github.com/Dongxiem/carrotCache/carrotcache/lru"
	"sync"
)

// cache：主要添加互斥锁来进行并发控制
type Cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	CacheBytes int64
}

// add：键值对添加
func (c *Cache) Add(key string, value byteview.ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 懒加载，进行实例化 lru
	// 一个对象的延迟初始化意味着该对象的创建将会延迟至第一次使用该对象时。主要用于提高性能，并减少程序内存要求
	if c.lru == nil {
		c.lru = lru.New(c.CacheBytes, nil)
	}
	// 已经实例化了之后将数据进行添加进 lru
	c.lru.Add(key, value)
}

// get：根据键得到值
func (c *Cache) Get(key string) (value byteview.ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	// 去 lru 当中找，找到则返回 ByteView 的只读数据
	if v, ok := c.lru.Get(key); ok {
		return v.(byteview.ByteView), ok
	}
	return
}
