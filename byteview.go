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
	"bytes"
	"errors"
	"io"
	"strings"
)

// A ByteView holds an immutable view of bytes.
// Internally it wraps either a []byte or a string,
// but that detail is invisible to callers.
// A ByteView 包含一个不改变字节视图
// 在内部包装了[]byte或字符串
// 但是调用者看不到该细节
//
// A ByteView is meant to be used as a value type, not
// a pointer (like a time.Time).
// A ByteView 应该被当做值来使用，而不是指针
type ByteView struct {
	// If b is non-nil, b is used, else s is used.
	b []byte
	s string
}

// Len returns the view's length.
// Len 返回view的长度
func (v ByteView) Len() int {
	if v.b != nil {
		return len(v.b)
	}
	return len(v.s)
}

// ByteSlice returns a copy of the data as a byte slice.
// ByteSlice 返回一份[]byte类型的view值的拷贝
func (v ByteView) ByteSlice() []byte {
	if v.b != nil {
		return cloneBytes(v.b)
	}
	return []byte(v.s)
}

// String returns the data as a string, making a copy if necessary.
// String 返回一份string类型的view值的拷贝
func (v ByteView) String() string {
	if v.b != nil {
		return string(v.b)
	}
	return v.s
}

// At returns the byte at index i.
// At 返回第i个byte
func (v ByteView) At(i int) byte {
	if v.b != nil {
		return v.b[i]
	}
	return v.s[i]
}

// Slice slices the view between the provided from and to indices.
// Slice 返回从索引from到to的切分结果
func (v ByteView) Slice(from, to int) ByteView {
	if v.b != nil {
		return ByteView{b: v.b[from:to]}
	}
	return ByteView{s: v.s[from:to]}
}

// SliceFrom slices the view from the provided index until the end.
// SliceFrom 返回索引到结尾到切分结果
func (v ByteView) SliceFrom(from int) ByteView {
	if v.b != nil {
		return ByteView{b: v.b[from:]}
	}
	return ByteView{s: v.s[from:]}
}

// Copy copies b into dest and returns the number of bytes copied.
// Copy 拷贝一份view到dest
func (v ByteView) Copy(dest []byte) int {
	if v.b != nil {
		return copy(dest, v.b)
	}
	return copy(dest, v.s)
}

// Equal returns whether the bytes in b are the same as the bytes in
// b2.
// Equal 相等判断，如果[]bytes为空，则判断字符串
func (v ByteView) Equal(b2 ByteView) bool {
	if b2.b == nil {
		return v.EqualString(b2.s)
	}
	return v.EqualBytes(b2.b)
}

// EqualString returns whether the bytes in b are the same as the bytes
// in s.
// EqualString 比较字符串是否相等
func (v ByteView) EqualString(s string) bool {
	if v.b == nil {
		// 如果b为nil，则比较s是否相等
		return v.s == s
	}
	// l为view的长度
	l := v.Len()
	// 如果长度不相等，则返回false
	if len(s) != l {
		return false
	}
	// 比较[]byte b中每一个byte是否和string s的每一个字节是否相等
	for i, bi := range v.b {
		if bi != s[i] {
			return false
		}
	}
	return true
}

// EqualBytes returns whether the bytes in b are the same as the bytes
// in b2.
// EqualBytes 比较[]bytes是否相等
func (v ByteView) EqualBytes(b2 []byte) bool {
	// b不为nil，则通过equal方法比较
	if v.b != nil {
		return bytes.Equal(v.b, b2)
	}
	l := v.Len()
	// 比较长度，不相等则返回false
	if len(b2) != l {
		return false
	}
	// 比较每一个byte
	for i, bi := range b2 {
		if bi != v.s[i] {
			return false
		}
	}
	return true
}

// Reader returns an io.ReadSeeker for the bytes in v.
// Reader 返回bytes包或string包的*Reader类型，该struct实现了io.ReadSeeker接口
func (v ByteView) Reader() io.ReadSeeker {
	if v.b != nil {
		return bytes.NewReader(v.b)
	}
	return strings.NewReader(v.s)
}

// ReadAt implements io.ReaderAt on the bytes in v.
// ReadAt 实现了 io.ReaderAt 接口
func (v ByteView) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, errors.New("view: invalid offset")
	}
	if off >= int64(v.Len()) {
		return 0, io.EOF
	}
	n = v.SliceFrom(int(off)).Copy(p)
	if n < len(p) {
		err = io.EOF
	}
	return
}

// WriteTo implements io.WriterTo on the bytes in v.
// WriteTo 实现了 io.WriteTo接口
func (v ByteView) WriteTo(w io.Writer) (n int64, err error) {
	var m int
	if v.b != nil {
		m, err = w.Write(v.b)
	} else {
		m, err = io.WriteString(w, v.s)
	}
	if err == nil && m < v.Len() {
		err = io.ErrShortWrite
	}
	n = int64(m)
	return
}
