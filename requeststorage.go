package scrapingo

import (
	"fmt"
	"sync"
)

type RequestStorage interface {
	//添加Request至Storage
	PushRequest(*Request)
	//從Storage取得Request
	PullRequest() *Request
	//返回當前儲存的個數
	Size() int
}

//已LinkedQueue的形式進行記憶體儲存
type InMemoryRequestQueue struct {
	//儲存上限
	MaxSize int
	//當前容量
	size int
	//Queue的頭指針 Pull時先從frist開始取
	frist *RequestNode
	//Queue的尾指針 Push從last以後開始添加
	last *RequestNode

	rw *sync.RWMutex
}

type RequestNode struct {
	Request *Request
	next    *RequestNode
}

//scrapingo默認使用 儲存上限為100000
func DefaultStorage() *InMemoryRequestQueue {
	return &InMemoryRequestQueue{MaxSize: 100000, rw: &sync.RWMutex{}}
}

//實現 RequestStorage interface的 PushRequest(*Request)
//傳入Request並將其設為last 當超過上限時進行Panic
func (r *InMemoryRequestQueue) PushRequest(req *Request) {
	r.rw.Lock()
	defer r.rw.Unlock()
	if r.MaxSize > 0 && r.MaxSize <= r.size {
		panic(ErrOverMaxRequestStorage.Error())
	}
	i := &RequestNode{Request: req}
	if r.frist == nil {
		r.frist = i
	} else {
		r.last.next = i
	}
	r.last = i
	r.size++
}

//實現 RequestStorage interface的 Size()int
//返回當前儲存容量
func (r *InMemoryRequestQueue) Size() int {
	r.rw.RLock()
	defer r.rw.RUnlock()
	return r.size
}

//實現 RequestStorage interface的 PullRequest()*Request
//返回當前frist的Request當size為0時返回nil
func (r *InMemoryRequestQueue) PullRequest() *Request {
	r.rw.Lock()
	defer r.rw.Unlock()
	if r.size == 0 {
		return nil
	}
	frist := r.frist.Request
	r.frist = r.frist.next
	r.size--
	return frist
}
func (r *InMemoryRequestQueue) String() string {
	return fmt.Sprintf(
		"RequestStorage:"+
			"\n\t\t|-Type:%T\n\t\t|-Size:%d\n\t\t|-MaxSize:%d",
		r, r.Size(), r.MaxSize,
	)
}
