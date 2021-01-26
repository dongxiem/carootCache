## carrotCache 特性

- 使用 `LRU` 缓存策略，并添加 `sync.Mutex` 互斥锁，实现 `LRU` 缓存并发控制；
- 使用一致性哈希（`consistent hashing`）选择节点，实现负载均衡；
- 使用 `singleflight` 缓存过滤机制，防止缓存击穿；
- 支持多节点互备热数据，避免频繁通过网络从远程节点获取数据；
- 建立基于 `HTTP` 的通信机制，实现缓存节点间通信；
- 支持 `Protobuf` 优化节点间二进制通信，提高效率；

## 项目框架

![img](https://camo.githubusercontent.com/c12f7037142aaf93fa4cbbbe953b232d24f7221a357a0345f74ed9bcab435922/687474703a2f2f7374617469632e696d6c67772e746f702f626c6f672f32303230303630322f476c6337614d7279536971462e706e673f696d616765736c696d)



## Example



```go
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
	"Lily": "589",
	"Pity":  "567",
}

func createGroup() *carrotcache.Group {
	return carrotcache.NewGroup("scores", 2<<10, carrotcache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func startCacheServer(addr string, addrs []string, cache *carrotcache.Group) {
	peers := h.NewHTTPPool(addr)
	peers.Set(addrs...)
	cache.RegisterPeers(peers)
	log.Println("carrotCache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(apiAddr string, cache *carrotcache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := cache.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "carrotCache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8051: "http://localhost:8001",
		8052: "http://localhost:8002",
		8053: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}
	cache := createGroup()
	if api {
		go startAPIServer(apiAddr, cache)
	}
	startCacheServer(addrMap[port], addrs, cache)
}
```















