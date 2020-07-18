package carrotCache

import pb "carrotCache/gocachepb"

// 主要是抽象出 2 个接口

// PeerPicker is the interface that must be implemented to locate
// the peer that owns a specific key.
// 该方法用于根据传入的 key 选择相应节点 PeerGetter
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter is the interface that must be implemented by a peer.
// 该法用于从对应 group 查找缓存值。
type PeerGetter interface {
	// 第二个参数使用 gocachepb.pb.go 中的数据类型
	Get(in *pb.Request, out *pb.Response) error
}
