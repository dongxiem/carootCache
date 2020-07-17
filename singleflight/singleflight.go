package singleflight

import "sync"

// call is an in-flight or completed Do call
// 代表正在进行中，或已经结束的请求。使用 sync.WaitGroup 锁避免重入
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group represents a class of work and forms a namespace in which
// units of work can be executed with duplicate suppression.
// Group 是 singleflight 的主数据结构，管理不同 key 的请求(call)
type Group struct {
	mu sync.Mutex       // protects m
	m  map[string]*call // lazily initialized
}

// Do executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a
// time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
// Do 的作用就是，针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，等待 fn 调用结束了，返回返回值或错误。
// 并发协程之间不需要消息传递，非常适合 sync.WaitGroup。
// 	wg.Add(1) 锁加1。
// 	wg.Wait() 阻塞，直到锁被释放。
// 	wg.Done() 锁减1
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	// g.mu 是保护 Group 的成员变量 m 不被并发读写而加上的锁
	g.mu.Lock()
	// 延迟初始化的目的很简单，提高内存使用效率
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait() // wg.Wait() 阻塞，直到锁被释放。
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)  // 发起请求前加锁
	g.m[key] = c // 添加到 g.m，表明 key 已经有对应的请求在处理
	g.mu.Unlock()

	c.val, c.err = fn() // 调用 fn，发起请求
	c.wg.Done()         // 请求结束

	g.mu.Lock()
	delete(g.m, key) // 更新 g.m
	g.mu.Unlock()

	return c.val, c.err // 返回结果
}
