package concurrentcache

import (
	"github.com/Dongxiem/carrotCache/byteview"
	"github.com/Dongxiem/carrotCache/lru"
	"sync"
)

// cache：主要添加互斥锁来进行并发控制
type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

// add：键值对添加
func (c *cache) add(key string, value byteview.ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 懒加载，进行实例化 lru
	// 一个对象的延迟初始化意味着该对象的创建将会延迟至第一次使用该对象时。主要用于提高性能，并减少程序内存要求
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	// 已经实例化了之后将数据进行添加进lru
	c.lru.Add(key, value)
}

// get：根据键得到值
func (c *cache) get(key string) (value byteview.ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	// 去lru当中找，找到则返回ByteView的只读数据
	if v, ok := c.lru.Get(key); ok {
		return v.(byteview.ByteView), ok
	}
	return
}
