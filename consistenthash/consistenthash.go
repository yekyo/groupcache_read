/*
Copyright 2013 Google Inc.

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

// Package consistenthash provides an implementation of a ring hash.
//这个包是一致性hash算法的实现
package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// hash是一个函数类型，形参是字节切片，返回无符号32位证书0 - 2^32-1
type Hash func(data []byte) uint32

// Map类型 第一个参数是hash函数 replicas 每一个cache节点的副本数
type Map struct {
	// Hash 上面定义的hash函数
	hash     Hash
	// replicas cache节点的副本数，虚拟节点
	replicas int
	// keys 包含所有节点的hash key，包括虚拟节点以及真实节点
	keys     []int // Sorted
	// hashMap key与服务器的映射关系
	hashMap  map[int]string
}

// 第一个形参为副本数，第二个为hash函数 返回map结构体指针
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		// 默认hash函数指定
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// IsEmpty returns true if there are no items available.
// keys非空检查
func (m *Map) IsEmpty() bool {
	return len(m.keys) == 0
}

// Add adds some keys to the hash.
// 添加cache服务器 key可以采用cache服务器ip例如 192.168.0.1,192.168.0.2
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		// 根据副本数量，添加多个节点
		for i := 0; i < m.replicas; i++ {
			// hash函数参数为编号i连接key
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			// 多个节点映射到一个cache服务器
			m.hashMap[hash] = key
		}
	}
	// 升序排序这个int切片
	sort.Ints(m.keys)
}

// Get gets the closest item in the hash to the provided key.
// key为缓存数据key
// 返回值为最近缓存服务器的key
func (m *Map) Get(key string) string {
	if m.IsEmpty() {
		return ""
	}
	// 通过key计算一个hash值，对应到hash环上的一个点
	hash := int(m.hash([]byte(key)))

	// Binary search for appropriate replica.
	// 二分查找 满足m.keys[i] >= hash的i的最小值
	// ring上顺时针方向离key最近的i i为缓存节点的key
	// 如果查找不到，默认返回len(m.keys)
	idx := sort.Search(len(m.keys), func(i int) bool { return m.keys[i] >= hash })

	// Means we have cycled back to the first replica.
	// 如果idx为len(m.keys)，则idx为0
	if idx == len(m.keys) {
		idx = 0
	}

	// idx为查找到的cache节点key
	// hashmap中存储节点与真实cache服务器的映射关系
	// 这里的返回值为真实服务器
	return m.hashMap[m.keys[idx]]
}
