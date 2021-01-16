package main


import (
	"flag"
	"fmt"
	"github.com/Dongxiem/carrotCache/carrotcache"
	h "github.com/Dongxiem/carrotCache/carrotcache/http"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
	"Lily": "589",
	"Pity":  "567",
}

// createGroup: 创建 Group
func createGroup() *carrotcache.Group {
	// 注意：carrotCache.GetterFunc 是回调函数。
	return carrotcache.NewGroup("scores", 2<<10, carrotcache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// startCacheServer： 开启 Cache 服务
func startCacheServer(addr string, addrs []string, cache *carrotcache.Group) {
	// 根据传递进来的地址 addr 创建一个新的 HTTP 池
	peers := h.NewHTTPPool(addr)
	// 对 peers 添加地址，该 addrs 是一串地址，为字符串切片
	peers.Set(addrs...)
	// 并且在 cache 中进行 peers 的注册
	cache.RegisterPeers(peers)
	log.Println("carrotCache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// startAPIServer： 开启 API 服务
func startAPIServer(apiAddr string, cache *carrotcache.Group) {
	// 进行 http.Handle 处理
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// 通过 URL 的 Query() 方法去得到 "key" 键所对应的具体键值
			key := r.URL.Query().Get("key")
			// 然后去 cache 当中得到对应的 value
			view, err := cache.Get(key)
			// 此时发生 err 则对应的是：内部服务器（HTTP-Internal Server Error）错误
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// 设置返回头部，置内容类型为："application/octet-stream"
			// 这是应用程序文件的默认值。意思是 未知的应用程序文件，浏览器一般不会自动执行或询问执行。
			w.Header().Set("Content-Type", "application/octet-stream")
			// 然后返回原始数据的拷贝值
			w.Write(view.ByteSlice())

		}))
	// 日志打印
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8051, "carrotCache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9994"
	addrMap := map[int]string{
		8051: "http://localhost:8051",
		8052: "http://localhost:8052",
		8053: "http://localhost:8053",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}
	cache := createGroup()
	if api {
		//带 api 参数的就是本机 self
		go startAPIServer(apiAddr, cache)
	}
	startCacheServer(addrMap[port], addrs, cache)
}
