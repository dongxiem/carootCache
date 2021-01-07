package carrotCache

import (
	"fmt"
	"github.com/Dongxiem/carrotCache/byteview"
	pb "github.com/Dongxiem/carrotCache/cachepb"
	concurrentcache "github.com/Dongxiem/carrotCache/concurrentcache"
	peers "github.com/Dongxiem/carrotCache/peers"
	"github.com/Dongxiem/carrotCache/singleflight"
	"log"
	"sync"
)

// 一个 Group 可以认为是一个缓存的命名空间，主要负责与外部交互，控制缓存存储和获取的主流程
type Group struct {
	name      string      // 每个 Group 拥有一个唯一的名称 name
	getter    Getter      // 缓存未命中时获取源数据的回调(callback)
	mainCache concurrentcache.Cache // 一开始实现的并发缓存
	peers     peers.PeerPicker
	// 使用 singleflight.Group 来确保每个 key 仅被提取一次
	loader *singleflight.Group	// 用于防止缓存击穿
}

// Getter：回调接口定义，只包含一个方法 Get
// 既能够将普通的函数类型（需类型转换）作为参数，也可以将结构体作为参数，使用更为灵活，可读性也更好，这就是接口型函数的价值。
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc：定义函数类型，GetterFunc 参数和返回值与 Getter 中 Get 方法是一致的。
type GetterFunc func(key string) ([]byte, error)

// Get：GetterFunc 还定义了 Get 方式，并在 Get 方法中调用自己，这样就实现了接口 Getter。
// 所以 GetterFunc 是一个实现了接口的函数类型，简称为接口型函数。
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group) // 将所有新生成的 Group 的指针及其对应的名字存储在全局变量 groups 中
)

// NewGroup： 创建一个新的Group实例
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	// 回调函数为空则报错
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	// 进行 Group 的注册
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: concurrentcache.Cache{CacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// GetGroup： 返回先前使用 NewGroup 创建的命名组，如果没有这样的组，则为 nil
func GetGroup(name string) *Group {
	// 使用了只读锁 RLock()，因为不涉及任何冲突变量的写操作
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get： 通过 key 去 cache 取相对应的 value
func (g *Group) Get(key string) (byteview.ByteView, error) {
	// 如果 key为空，返回空的 ByteView，然后再返回一个 Error
	if key == "" {
		return byteview.ByteView{}, fmt.Errorf("key is required")
	}

	// 从 mainCache 中查找缓存，如果存在则缓存命中，并且返回缓存值
	if v, ok := g.mainCache.Get(key); ok {
		log.Println("[GoCache] hit")
		return v, nil
	}
	// 如果缓存中不存在，则调用 load 方法去远程节点进行数据的获取，实在没有再去数据库进行数据获取，最后添加到缓存当中。
	return g.load(key)
}

// RegisterPeers：该方法实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(peers peers.PeerPicker) {
	// 如果原来的 group 已存在 peers，即此时重复注册，则会 panic
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	// 进行注入
	g.peers = peers
}

// load：进行数据获取，尝试本地节点或者其他节点进行缓存数据的获取，都获取不到再去本地数据库获取。
func (g *Group) load(key string) (value byteview.ByteView, err error) {
	// n个协程同时调用了g.Do，fn中的逻辑只会被一个协程执行，这里是实现了 singleflight 的内容，防止缓存穿透。
	// 使用 g.loader.Do进行包装，确保了并发场景下针对相同的 key，load 过程只会调用一次。
	// 使用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer() 从远程获取
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		// 下面为 fn 方法的具体实现，该方法在多个协程请求的情况下只会执行一次。
		// 首先判断 group.peers 缓存节点是否为空，如果不为空，则根据 key 找到相对应的缓存节点 peer
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				// 去指定的缓存节点 Peer 根据 key 进行数据的获取请求，并得到数据 value
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		// 若是本机节点或远程节点获取失败，则回退到 getLocally()
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(byteview.ByteView), nil
	}
	return
}

// populateCache：添加数据进cache
func (g *Group) populateCache(key string, value byteview.ByteView) {
	// 添加到当前group对应的cache中
	g.mainCache.Add(key, value)
}

// getLocally：缓存不存在时，调用回调函数获取源数据
func (g *Group) getLocally(key string) (byteview.ByteView, error) {
	// 调用用户回调函数 g.getter.Get() 获取源数据
	bytes, err := g.getter.Get(key)
	if err != nil {
		return byteview.ByteView{}, err
	}
	// 通过 ByteView 中的 cloneBytes 方法进行拷贝数据赋值给 value，不要影响到原数据
	value := byteview.ByteView{B: byteview.CloneBytes(bytes)}
	// 并且将源数据添加到缓存 mainCache 中，下次再进行 key 的获取就可以从缓存中查找到了
	g.populateCache(key, value)
	return value, nil
}

// getFromPeer：使用实现了 PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值。
func (g *Group) getFromPeer(peer peers.PeerGetter, key string) (byteview.ByteView, error) {
	// 首先进行 Request 的注册
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	// res 初始为 {}
	res := &pb.Response{}
	// 根据 req 获取相对应的 res
	err := peer.Get(req, res)
	if err != nil {
		return byteview.ByteView{}, err
	}
	// 将该 res.Value 转为 []byte 并且进行返回
	return byteview.ByteView{B: res.Value}, nil
}
