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

// Package singleflight provides a duplicate function call suppression
// singleflight 提供重复调用抑制机制
// mechanism.
package singleflight

import "sync"

// call is an in-flight or completed Do call
// call 是在执行的或者已经完成的Do过程
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group represents a class of work and forms a namespace in which
// units of work can be executed with duplicate suppression.
// Group 代表一类工作并组成一个命名空间，其中的工作单元可以抑制重复执行
type Group struct {
	mu sync.Mutex       // protects m
	m  map[string]*call // lazily initialized
}

// Do executes and returns the results of the given function, making
// sure that only one execution is in-flight for a given key at a
// time. If a duplicate comes in, the duplicate caller waits for the
// original to complete and receives the same results.
// Do 接收函数，执行并返回执行结果
// 确保执行过程中，只有一个key在同一时间执行
// 如果是重复调用，会等待最原始的调用完成，接收到相同的结果
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	// 如果g.m为尚未初始化，则初始化
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 如果key存在同名调用，则等待原始调用完成，然后返回其val和err
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		// 等待goroutine执行完成，call中存储来执行的结果val和err
		return c.val, c.err
	}
	// 拿到call结构体的指针
	c := new(call)
	// 因为一个函数只执行一次，所以wg.add(1), 同一key的fn函数只调用一次
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	// 函数调用完成，返回结果和错误信息
	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	// 执行完成，删除对应key
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
