/*
Copyright 2012 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package groupcache

import (
	"errors"

	"github.com/golang/protobuf/proto"
)

// A Sink receives data from a Get call.
// 一个Sink从Get调用获取到数据
//
// Implementation of Getter must call exactly one of the Set methods
// on success.
// 实现Getter必须调用下面的某一个特定的Set方法
type Sink interface {
	// SetString sets the value to s.
	SetString(s string) error

	// SetBytes sets the value to the contents of v.
	// The caller retains ownership of v.
	SetBytes(v []byte) error

	// SetProto sets the value to the encoded version of m.
	// The caller retains ownership of m.
	SetProto(m proto.Message) error

	// view returns a frozen view of the bytes for caching.
	// view 返回一个冻结的字节视图用以缓存
	view() (ByteView, error)
}

// 克隆byte切片
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}


func setSinkView(s Sink, v ByteView) error {
	// A viewSetter is a Sink that can also receive its value from
	// a ByteView. This is a fast path to minimize copies when the
	// item was already cached locally in memory (where it's
	// cached as a ByteView)
	// 一个viewSetter是一个Sink可以从ByteView中接收它的值
	// 这是在item已经缓存到本地内存中（当它作为ByteView被缓存）时，将副本减少到最小到快速途径

	// 定义一个viewSetter的接口类型
	type viewSetter interface {
		// 定义setView方法参数以及返回值
		setView(v ByteView) error
	}
	// 判断s是否实现了viewSetter接口
	if vs, ok := s.(viewSetter); ok {
		// 实现了则调用setView方法
		return vs.setView(v)
	}
	// 如果s没有实现viewSetter接口，v.b不为ni则调用SetBytes方法
	if v.b != nil {
		return s.SetBytes(v.b)
	}
	// 调用SetString方法，并返回
	return s.SetString(v.s)
}

// StringSink returns a Sink that populates the provided string pointer.
// StringSink 返回一个Sink,该Sink使用提供的字符串指针填充
func StringSink(sp *string) Sink {
	return &stringSink{sp: sp}
}

// sp 是string pointer字符串指针
// 两个成员，一个为字符串指针、一个为ByteView类型
type stringSink struct {
	sp *string
	v  ByteView
	// TODO(bradfitz): track whether any Sets were called.
}

// 返回stringSink的ByteView以及错误信息
func (s *stringSink) view() (ByteView, error) {
	// TODO(bradfitz): return an error if no Set was called
	return s.v, nil
}

// 设置sp和v.s，也就是stringSink中字符串相关属性
func (s *stringSink) SetString(v string) error {
	s.v.b = nil
	s.v.s = v
	*s.sp = v
	return nil
}

// 调用SetString方法，将v []byte转换为string并设置
func (s *stringSink) SetBytes(v []byte) error {
	return s.SetString(string(v))
}

// 从proto.Message中获取数据，写入stringSink的sp和v.b中
func (s *stringSink) SetProto(m proto.Message) error {
	b, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	s.v.b = b
	*s.sp = string(b)
	return nil
}

// ByteViewSink returns a Sink that populates a ByteView.
// ByteViewSink 返回一个Sink，该Sink使用提供的ByteView填充
func ByteViewSink(dst *ByteView) Sink {
	if dst == nil {
		panic("nil dst")
	}
	return &byteViewSink{dst: dst}
}

type byteViewSink struct {
	dst *ByteView

	// if this code ever ends up tracking that at least one set*
	// method was called, don't make it an error to call set
	// methods multiple times. Lorry's payload.go does that, and
	// it makes sense. The comment at the top of this file about
	// "exactly one of the Set methods" is overly strict. We
	// really care about at least once (in a handler), but if
	// multiple handlers fail (or multiple functions in a program
	// using a Sink), it's okay to re-use the same one.
	// 如果此代码最终跟踪了至少一组 Set* 方法被调用
	// 多次调用set方法不出错
	// larry的payload.go做到了，这很有意义
	// 此文件的顶部注释 "exactly one of the Set methods" 过于严格
	// 我们真的关心至少一次（在处理程序中）
	// 但是如果多个处理程序失败（或者在一个程序中多个函数使用一个Sink）
	// 可以重用相同的结果
}

// 设置view
func (s *byteViewSink) setView(v ByteView) error {
	*s.dst = v
	return nil
}

// 获取ByteView
func (s *byteViewSink) view() (ByteView, error) {
	return *s.dst, nil
}

// 设置byteViewSink的dst，参数类型为proto.Message
func (s *byteViewSink) SetProto(m proto.Message) error {
	b, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	*s.dst = ByteView{b: b}
	return nil
}

// 设置byteViewSink的dst，参数类型为字节切片[]byte
func (s *byteViewSink) SetBytes(b []byte) error {
	*s.dst = ByteView{b: cloneBytes(b)}
	return nil
}

// 设置byteViewSink的dst，参数类型为string
func (s *byteViewSink) SetString(v string) error {
	*s.dst = ByteView{s: v}
	return nil
}

// ProtoSink returns a sink that unmarshals binary proto values into m.
// ProtoSink 返回一个sink 使用proto.Message类型的m 初始化protoSink的dst
func ProtoSink(m proto.Message) Sink {
	return &protoSink{
		dst: m,
	}
}

// 定义一个protoSink的结构体
type protoSink struct {
	dst proto.Message // authoritative value
	typ string

	v ByteView // encoded
}

// 获取ByteView
func (s *protoSink) view() (ByteView, error) {
	return s.v, nil
}

// 将s.dst反序列化后给b，并且复制b赋值给protoSink中的ByteView的b
func (s *protoSink) SetBytes(b []byte) error {
	err := proto.Unmarshal(b, s.dst)
	if err != nil {
		return err
	}
	s.v.b = cloneBytes(b)
	s.v.s = ""
	return nil
}

// 参数为string，将s.dst反序列化后给b，并且b赋值给protoSink中的ByteView的b
func (s *protoSink) SetString(v string) error {
	b := []byte(v)
	err := proto.Unmarshal(b, s.dst)
	if err != nil {
		return err
	}
	s.v.b = b
	s.v.s = ""
	return nil
}

// 将m序列化后，赋值给protoSink的ByteView的b
func (s *protoSink) SetProto(m proto.Message) error {
	b, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	// TODO(bradfitz): optimize for same-task case more and write
	// right through? would need to document ownership rules at
	// the same time. but then we could just assign *dst = *m
	// here. This works for now:
	// 对吗？需要同时记录所有权规则
	// 但是我们可以在这里赋值 *dst = *m
	// 目前适用：
	err = proto.Unmarshal(b, s.dst)
	if err != nil {
		return err
	}
	s.v.b = b
	s.v.s = ""
	return nil
}

// AllocatingByteSliceSink returns a Sink that allocates
// a byte slice to hold the received value and assigns
// it to *dst. The memory is not retained by groupcache.
// AllocationByteSliceSink 返回一个Sink,该Sink分配一个字节切片来保存接收到的值，
// 并将其分配给*dst。groupcache不保留内存
func AllocatingByteSliceSink(dst *[]byte) Sink {
	return &allocBytesSink{dst: dst}
}

// 定义一个allocBytesSink的结构体
type allocBytesSink struct {
	dst *[]byte
	v   ByteView
}

// 获取ByteView
func (s *allocBytesSink) view() (ByteView, error) {
	return s.v, nil
}

// 设置ByteView，参数类型为ByteVIew
func (s *allocBytesSink) setView(v ByteView) error {
	if v.b != nil {
		*s.dst = cloneBytes(v.b)
	} else {
		*s.dst = []byte(v.s)
	}
	s.v = v
	return nil
}

// 设置allocByteSink的dst和ByteView，参数类型为proto.Message
func (s *allocBytesSink) SetProto(m proto.Message) error {
	b, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	return s.setBytesOwned(b)
}

// 设置allocByteSink的dst和ByteView，参数类型为字节切片
func (s *allocBytesSink) SetBytes(b []byte) error {
	return s.setBytesOwned(cloneBytes(b))
}

// 参数为b字节切片，设置allocBytesSink的dst和ByteView
func (s *allocBytesSink) setBytesOwned(b []byte) error {
	if s.dst == nil {
		return errors.New("nil AllocatingByteSliceSink *[]byte dst")
	}
	*s.dst = cloneBytes(b) // another copy, protecting the read-only s.v.b view 额外的拷贝，保护s.v.b ByteView的只读性
	s.v.b = b
	s.v.s = ""
	return nil
}

// 参数类型string，设置allocByteSink的ByteView的属性s.v.s
func (s *allocBytesSink) SetString(v string) error {
	if s.dst == nil {
		return errors.New("nil AllocatingByteSliceSink *[]byte dst")
	}
	*s.dst = []byte(v)
	s.v.b = nil
	s.v.s = v
	return nil
}

// TruncatingByteSliceSink returns a Sink that writes up to len(*dst)
// bytes to *dst. If more bytes are available, they're silently
// truncated. If fewer bytes are available than len(*dst), *dst
// is shrunk to fit the number of bytes available.
// TruncatingByteSliceSink 返回一个Sink，它将最多len(*dst)个字节写入*dst
// 如果更多字节可用，它们会被静默截断
// 如果可用字节少于len(*dst),则将缩小*dst以适合可用字节数
func TruncatingByteSliceSink(dst *[]byte) Sink {
	return &truncBytesSink{dst: dst}
}

// 定义一个truncBytesSink结构体
type truncBytesSink struct {
	dst *[]byte
	v   ByteView
}

// 获取truncBytesSink的ByteView
func (s *truncBytesSink) view() (ByteView, error) {
	return s.v, nil
}

// 参数类型为proto.Message,设置truncBytesSink的ByteView
func (s *truncBytesSink) SetProto(m proto.Message) error {
	b, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	return s.setBytesOwned(b)
}

// 参数类型为字节切片，设置truncBytesSink的ByteView
func (s *truncBytesSink) SetBytes(b []byte) error {
	return s.setBytesOwned(cloneBytes(b))
}

// 将b字节切片拷贝到*s.dst，如果len(*s.dst)>拷贝的字节长度，*s.dst将被截断
// 将b赋值给truncByteSink的ByteView的s.v.b
func (s *truncBytesSink) setBytesOwned(b []byte) error {
	if s.dst == nil {
		return errors.New("nil TruncatingByteSliceSink *[]byte dst")
	}
	n := copy(*s.dst, b)
	if n < len(*s.dst) {
		*s.dst = (*s.dst)[:n]
	}
	s.v.b = b
	s.v.s = ""
	return nil
}

// 将v字符串拷贝到*s.dst，如果len(*s.dst)>拷贝到长度,*s.dst将被截断
// 将v赋值给truncByteSink的ByteView的s.v.s
func (s *truncBytesSink) SetString(v string) error {
	if s.dst == nil {
		return errors.New("nil TruncatingByteSliceSink *[]byte dst")
	}
	n := copy(*s.dst, v)
	if n < len(*s.dst) {
		*s.dst = (*s.dst)[:n]
	}
	s.v.b = nil
	s.v.s = v
	return nil
}
