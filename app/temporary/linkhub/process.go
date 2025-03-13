package linkhub

import (
	"reflect"

	"github.com/vela-ssoc/ssoc-broker/app/temporary"
)

// process 处理器
type process struct {
	fn  reflect.Value // 执行的方法
	arg bool          // 是否带有参数
	ptr bool          // 参数是否是指针类型
	in  reflect.Type  // 入参反射类型
}

// Invoke 开始执行注册函数调用
func (p process) Invoke(conn *temporary.Conn, rec *temporary.Receive) error {
	rc := reflect.ValueOf(conn)
	var args []reflect.Value
	if !p.arg { // 没有其它参数的情况
		args = []reflect.Value{rc}
	} else {
		// val := p.in.Interface() 错误
		val := reflect.New(p.in).Interface()
		if err := rec.JSON(val); err != nil {
			return err
		}
		rin := reflect.ValueOf(val)
		if !p.ptr {
			rin = rin.Elem()
		}
		args = []reflect.Value{rc, rin}
	}

	defer func() { _ = recover() }()

	ret := p.fn.Call(args)[0]
	if ret.IsNil() {
		return nil
	}

	return ret.Interface().(error)
}
