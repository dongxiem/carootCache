## carrotCache 特性

- 使用 `LRU` 缓存策略，并添加 `sync.Mutex` 互斥锁，实现 `LRU` 缓存并发控制；
- 使用一致性哈希（`consistent hashing`）选择节点，实现负载均衡；
- 使用 `singleflight` 缓存过滤机制，防止缓存击穿；
- 支持多节点互备热数据，避免频繁通过网络从远程节点获取数据；
- 建立基于 `HTTP` 的通信机制，实现缓存节点间通信；
- 支持 `Protobuf` 优化节点间二进制通信，提高效率；

## 项目框架

![img](https://camo.githubusercontent.com/c12f7037142aaf93fa4cbbbe953b232d24f7221a357a0345f74ed9bcab435922/687474703a2f2f7374617469632e696d6c67772e746f702f626c6f672f32303230303630322f476c6337614d7279536971462e706e673f696d616765736c696d)

































