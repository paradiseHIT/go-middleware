package app

import (
	"container/list"
	"sync"
)

// 这个是queue的数据结构，由于contain/list是线程不安全的，需要加锁处理
type Queue struct {
	List list.List
	Lock sync.Mutex
}

// 入
func (this *Queue) Push(a interface{}) {
	defer this.Lock.Unlock()
	this.Lock.Lock()
	this.List.PushFront(a)
}

// 出
func (this *Queue) Pop() interface{} {
	defer this.Lock.Unlock()
	this.Lock.Lock()
	e := this.List.Back()
	if e != nil {
		this.List.Remove(e)
		return e.Value
	}
	return nil
}
func (this *Queue) Len() int {
	defer this.Lock.Unlock()
	this.Lock.Lock()
	return this.List.Len()
}
