package peers

import pb "github.com/Dongxiem/carrotCache/carrotcache/cachepb"

// PeerPicker：这是一个接口，根据传入的 key 选择相应节点 PeerGetter。
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter：这是一个接口，用于从对应 group 查找缓存值。
type PeerGetter interface {
	// 第二个参数使用 cachepb.pb.go 中的数据类型
	Get(in *pb.Request, out *pb.Response) error
}
