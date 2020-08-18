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

// Package lru implements an LRU cache.
//lru包实现LRU(Least Recently Used 最近最少使用)缓存算法
package lru

import "container/list"

// Cache is an LRU cache. It is not safe for concurrent access.
// Cache结构体是LRU cache算法，并发访问不安全
type Cache struct {
	// MaxEntries is the maximum number of cache entries before
	// an item is evicted. Zero means no limit.
	// 最大缓存数量，0代表无限制
	MaxEntries int

	// OnEvicted optionally specifies a callback function to be
	// executed when an entry is purged from the cache.
	// 缓存实体被清除时回调函数
	OnEvicted func(key Key, value interface{})

	// 双向链表
	ll    *list.List
	// map key为任意类型 value为链表节点指针
	cache map[interface{}]*list.Element
}

// A Key may be any value that is comparable. See http://golang.org/ref/spec#Comparison_operators
// 任何可比较的类型
type Key interface{}

// 记录结构体
type entry struct {
	key   Key
	value interface{}
}

// New creates a new Cache.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
// New 创建新的缓存实例
// 如果 maxEntries为零，则缓存没有限制，淘汰缓存由调用者完成
func New(maxEntries int) *Cache {
	return &Cache{
		MaxEntries: maxEntries,
		ll:         list.New(),
		cache:      make(map[interface{}]*list.Element),
	}
}

// Add adds a value to the cache.
// Add 往缓存中添加一个值
func (c *Cache) Add(key Key, value interface{}) {
	// 如果缓存没有初始化，则先初始化
	if c.cache == nil {
		c.cache = make(map[interface{}]*list.Element)
		c.ll = list.New()
	}
	// 如果key已经存在，则将记录移动到链表的头部，然后设置value
	if ee, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ee)
		ee.Value.(*entry).value = value
		return
	}
	// 如果key不存在,创建记录，并将记录移动到链表的头部,ele为链表节点指针
	ele := c.ll.PushFront(&entry{key, value})
	c.cache[key] = ele
	// 缓存最大数量，超过触发清理最旧记录
	if c.MaxEntries != 0 && c.ll.Len() > c.MaxEntries {
		c.RemoveOldest()
	}
}

// Get looks up a key's value from the cache.
// Get 根据key查找value
func (c *Cache) Get(key Key) (value interface{}, ok bool) {
	if c.cache == nil {
		return
	}
	// 如果key存在
	if ele, hit := c.cache[key]; hit {
		// 将记录移动至链表头部
		c.ll.MoveToFront(ele)
		// 返回记录的值
		return ele.Value.(*entry).value, true
	}
	return
}

// Remove removes the provided key from the cache.
// Remove 移除指定key的缓存记录
func (c *Cache) Remove(key Key) {
	if c.cache == nil {
		return
	}
	if ele, hit := c.cache[key]; hit {
		c.removeElement(ele)
	}
}

// RemoveOldest removes the oldest item from the cache.
// RemoveOldest 移除最旧的缓存记录
func (c *Cache) RemoveOldest() {
	if c.cache == nil {
		return
	}
	// else赋值为链表尾部节点指针
	ele := c.ll.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

// 移除记录节点
func (c *Cache) removeElement(e *list.Element) {
	// 移除链表中节点
	c.ll.Remove(e)
	// e.value是interface{}类型，通过类型断言转换为*entry类型 记录指针
	// entry结构体 包含key、value两个属性
	kv := e.Value.(*entry)
	delete(c.cache, kv.key)
	// 如果存在清除回调函数，触发清除回调
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}

// Len returns the number of items in the cache.
// 返回缓存记录数目
func (c *Cache) Len() int {
	if c.cache == nil {
		return 0
	}
	return c.ll.Len()
}

// Clear purges all stored items from the cache.
// 删除缓存所有记录
func (c *Cache) Clear() {
	// 如果清除回调函数存在，则清除节点时依次调用
	if c.OnEvicted != nil {
		for _, e := range c.cache {
			kv := e.Value.(*entry)
			c.OnEvicted(kv.key, kv.value)
		}
	}
	c.ll = nil
	c.cache = nil
}
