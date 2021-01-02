package carrotCache

import (
	"fmt"
	pb "github.com/Dongxiem/carrotCache/cachepb"
	"github.com/Dongxiem/carrotCache/consistenthash"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
)

// 分布式缓存需要实现节点间通信，建立基于 HTTP 的通信机制是比较常见和简单的做法。
// 如果一个节点启动了 HTTP 服务，那么这个节点就可以被其他节点访问。
// 主要为基于http提供被其他节点访问的能力
const (
	defaultBasePath = "/_carrotcache/"
	defaultReplicas = 50
)

// HTTPPool 既具备了提供 HTTP 服务的能力，也具备了根据具体的 key，创建 HTTP 客户端从远程节点获取缓存值的能力
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:8000"
	self     string              // 用来记录自己的地址，包括主机名/IP 和端口
	basePath string              // 作为节点间通讯地址的前缀，默认是 /_gocache/
	mu       sync.Mutex          // guards peers and httpGetters
	peers    *consistenthash.Map // 类型是一致性哈希算法的 Map，用来根据具体的 key 选择节点。

	// 映射远程节点与对应的 httpGette
	// 每一个远程节点对应一个 httpGetter，因为 httpGetter 与远程节点的地址 baseURL 有关
	// keyed by e.g. "http://10.0.0.2:8008"
	httpGetters map[string]*httpGetter
}

// NewHTTPPool : 为每个节点初始化HTTP池
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log ：日志打印
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP ：进行所有http请求的处理
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 首先判断访问路径的前缀是否是 basePath，不是返回错误
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// 约定访问路径格式为 /<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	// 如果请求长度不为2则报错
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// 分别将groupName和key提取出来
	groupName := parts[0]
	key := parts[1]
	// 通过 groupname 得到 group 实例
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	// 再使用 group.Get(key) 获取缓存数据
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// 将得到的value作为proto消息写入响应主体
	// 使用 proto.Marshal() 编码 HTTP 响应
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	// 最终使用 w.Write() 将缓存值作为 httpResponse 的 body 返回
	w.Write(body)
}

// Set ：该方法实例化了一致性哈希算法，并且添加了传入的节点
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 进行实例化
	p.peers = consistenthash.New(defaultReplicas, nil)
	// 添加的节点进行补充到后面
	p.peers.Add(peers...)
	// 为每一个节点创建了一个 HTTP 客户端 httpGetter
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer : 根据传入的key挑选一个节点
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

type httpGetter struct {
	baseURL string
}

// Get : 数据获取
func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL, // baseURL 表示将要访问的远程节点的地址
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	// 使用 http.Get() 方式获取返回值
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	// 转换为 []bytes 类型
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}
	// 使用 proto.Unmarshal() 解码 HTTP 响应
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

var _ PeerGetter = (*httpGetter)(nil)
