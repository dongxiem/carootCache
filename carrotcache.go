package carrotCache

import (
	"fmt"
	pb "github.com/Dongxiem/carrotCache/cachepb"
	"github.com/Dongxiem/carrotCache/singleflight"
	"log"
	"sync"
)

// 主要负责与外部交互，控制缓存存储和获取的主流程
// 一个 Group 可以认为是一个缓存的命名空间
type Group struct {
	name      string      // 每个 Group 拥有一个唯一的名称 name
	getter    Getter      // 缓存未命中时获取源数据的回调(callback)
	mainCache cache.cache // 一开始实现的并发缓存
	peers     PeerPicker
	// use singleflight.Group to make sure that
	// each key is only fetched once
	loader *singleflight.Group	// 用于防止缓存击穿
}

// Getter回调接口定义
type Getter interface {
	Get(key string) ([]byte, error)
}

// 定义函数类型 GetterFunc，GetterFunc 通过函数实现Getter。
type GetterFunc func(key string) ([]byte, error)

// 获取实现Getter接口功能
// 回调函数 Get(key string)([]byte, error)，参数是 key，返回值是 []byte
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group) // 将所有新生成的group及其对应的名字存储在全局变量 groups 中
)

// NewGroup创建一个新的Group实例
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	// 回调函数为空则报错
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache.cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// GetGroup返回先前使用NewGroup创建的命名组，如果没有这样的组，则为nil
func GetGroup(name string) *Group {
	// 使用了只读锁 RLock()，因为不涉及任何冲突变量的写操作
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// 通过key去cache取相对应的value
func (g *Group) Get(key string) (ByteView, error) {
	// 如果key为空
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	// 从 mainCache 中查找缓存，如果存在则返回缓存值
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GoCache] hit")
		return v, nil
	}
	// 如果缓存不存在，则调用 load 方法
	return g.load(key)
}

// RegisterPeers registers a PeerPicker for choosing remote peer
// 实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 进行数据获取
func (g *Group) load(key string) (value ByteView, err error) {
	// each key is only fetched once (either locally or remotely)
	// regardless of the number of concurrent callers.
	// 使用 g.loader.Do进行包装，确保了并发场景下针对相同的 key，load 过程只会调用一次
	// 使用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer() 从远程获取
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		// 若是本机节点或远程获取失败，则回退到 getLocally()
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

// 添加数据进cache
func (g *Group) populateCache(key string, value ByteView) {
	// 添加到当前group对应的cache中
	g.mainCache.add(key, value)
}

// 缓存不存在时，调用回调函数获取源数据
func (g *Group) getLocally(key string) (ByteView, error) {
	// 调用用户回调函数 g.getter.Get() 获取源数据
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err

	}
	// 通过ByteView中的cloneBytes方法赋值一份数据并赋值给value
	value := ByteView{b: cloneBytes(bytes)}
	// 并且将源数据添加到缓存 mainCache 中
	g.populateCache(key, value)
	return value, nil
}

// 使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
