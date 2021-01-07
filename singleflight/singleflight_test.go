package singleflight

import (
	"testing"
)

// TestDo：测试 Do 的作用
func TestDo(t *testing.T) {
	var g Group
	// 其中第二个参数是一个函数，调用 Do 会进行运行该函数然后返回 interface{} 和 error。
	v, err := g.Do("key", func() (interface{}, error) {
		return "bar", nil
	})
	// 接着验证结果
	if v != "bar" || err != nil {
		t.Errorf("Do v = %v, error = %v", v, err)
	}
}
