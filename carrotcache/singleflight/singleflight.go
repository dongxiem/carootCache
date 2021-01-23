package singleflight

import "sync"

// call：代表正在进行中，或已经结束的请求。使用 sync.WaitGroup 避免重入
type call struct {
	wg  sync.WaitGroup	// 用于阻塞这个调用 call 的其他请求
	val interface{}		// 函数执行后的结果
	err error			// 函数执行后的 error
}


// Group 是 singleflight 的主数据结构，管理不同 key 的请求(call)
type Group struct {
	mu sync.Mutex       // 保护 m
	m  map[string]*call // 懒加载，键是 string，值是 call 结构体
}

// Do：执行并返回给定函数的结果，确保一次仅对给定键进行一次执行。
// 如果出现重复请求尽量，则重复的 caller 将等待原始请求完成并收到相同的结果。
// 并发协程之间不需要消息传递，非常适合 sync.WaitGroup。
// 	wg.Add(1) 计数加1。
// 	wg.Wait() 阻塞，直到资源被释放。
// 	wg.Done() 计数减1
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	// g.mu 是保护 Group 的成员变量 m 不被并发读写而加上的锁
	g.mu.Lock()
	// 进行延迟初始化，延迟初始化的目的很简单，提高内存使用效率
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 如果获取当前key的函数正在被执行，则阻塞等待执行中的，等待其执行完毕后获取它的执行结果
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		// 如果请求正在进行中，则等待，直到 wg 被释放。
		c.wg.Wait()
		// 请求结束，返回结果
		return c.val, c.err
	}
	c := new(call)
	// 发起请求前计数加一
	c.wg.Add(1)
	// 添加到 g.m，表明 key 已经有对应的请求在处理
	g.m[key] = c
	g.mu.Unlock()

	// 执行获取 key 的函数，并将结果赋值给这个 Call
	c.val, c.err = fn()
	// 请求结束
	c.wg.Done()

	g.mu.Lock()
	// 重新上锁，并将该 key 剔除，下一个 key 进来可以进行访问了
	delete(g.m, key)
	g.mu.Unlock()
	// 返回结果
	return c.val, c.err
}
