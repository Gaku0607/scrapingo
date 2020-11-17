package scrapingo

import (
	"context"
	"fmt"
)

type Scheduler interface {

	//提交Request至調度器進行儲存

	Submit(*Request)

	//傳入ThreadPoolSize以及
	//每個Thread所對應的RequestChanBuf 設置 ThreadPool

	ConfigPool(int, int)

	//傳入scrapingo.RequestStorage 設置 Shedler的Request儲存方式

	ConfigStorage(RequestStorage)

	//返回每個Thread所對應的RequestChan

	RequestChan() chan *Request

	//傳入完成信號 以及 Context 啟用調度器
	//調度器會將儲存的Request分配給每個Thread處理進行處理

	Run(context.Context, chan struct{})
}

type MultipleScheduler struct {

	//儲存所有Requset
	//參考（scrapingo.Storage）interface 的實現方式

	requestStorage RequestStorage

	//存放所有Thread所對應的 RequestChan

	threadPool []chan *Request

	//ThreadPool的大小

	poolsize int

	//紀錄ThreadPool當前所使用的Thread的位子

	ptr int
}

//實現了Sheduler interface 的 Submit(*Request)
//把Request提交至requestStorage
func (m *MultipleScheduler) Submit(r *Request) {
	m.requestStorage.PushRequest(r)
}

//實現了Sheduler interface 的 RequestChan()chan *Request
//返回每個Thread對應的RequestChan
func (m *MultipleScheduler) RequestChan() chan *Request {
	return m.pull()
}

//實現了Sheduler interface 的 ConfigPool(int,int)
//初始化ThreadPool 以及每個Thread所對應的RequestChan
func (m *MultipleScheduler) ConfigPool(size, buf int) {
	m.poolsize = size
	m.threadPool = make([]chan *Request, size)

	for i := 0; i < size; i++ {
		m.threadPool[i] = make(chan *Request, buf)
	}
}

//實現了Sheduler interface 的 ConfigStorage(Storage)
//當傳入的值為nil時 調用 DefaultStorage()
func (m *MultipleScheduler) ConfigStorage(q RequestStorage) {
	if q == nil {
		m.requestStorage = DefaultStorage()
	}
	m.requestStorage = q
}

//實現了Sheduler interface 的 Run(context.Context, chan struct{})
//調用Ctx的Canecl()時即可停止調度
//調度器會將儲存的Request分配給每個Thread處理進行處理
//該Thread當處理完畢時 會通過 complete chan 提交完成信號 告知Scheduler
//直到RequestStorage為空並且所有線程閒置時結束Run()
func (m *MultipleScheduler) Run(ctx context.Context, c chan struct{}) {
	go func(complete <-chan struct{}) {
		var active int
		var req *Request
		for {
			var activeThread chan *Request
			if m.IsEmpty() && active == 0 {
				m.closeThreadPool()
				return
			}
			if !m.IsEmpty() {
				activeThread = m.peek()
				req = m.requestStorage.PullRequest()
			}
		Loop:
			for {
				select {
				case activeThread <- req:
					active++
					for m.enqueue(activeThread, m.requestStorage.PullRequest()) {
						active++
					}
					break Loop
				case <-complete:
					active--
					if activeThread == nil && active == 0 {
						break Loop
					}
				case <-ctx.Done():
					m.closeThreadPool()
					return
				}
			}
		}

	}(c)
}

//查看RequestStorage是否為空
func (m *MultipleScheduler) IsEmpty() bool {
	return m.requestStorage.Size() == 0
}

//直到該Thread所對應的RequestChan Blocking為止不斷傳入Request
//當RequestChan Blocking時 移動ThreadPool的ptr
//並且將為未提交成功的Request存回RequestStorage中
func (m *MultipleScheduler) enqueue(c chan<- *Request, r *Request) bool {
	if r == nil {
		return false
	}
	select {
	case c <- r:
		return true
	default:
		m.Submit(r)
		m.next()
		return false
	}
}

//關閉ThreadPool中所有線程的Chan
func (m *MultipleScheduler) closeThreadPool() {
	for _, thread := range m.threadPool {
		close(thread)
	}
}

//返回ThreadPool中當前Thread所對應的RequestChan
//並且移動ptr
func (m *MultipleScheduler) pull() chan *Request {
	val := m.threadPool[m.ptr]
	m.ptr = (m.ptr + 1) % m.poolsize
	return val
}

//返回ThreadPool中當前Thread所對應的RequestChan
func (m *MultipleScheduler) peek() chan *Request {
	return m.threadPool[m.ptr]
}

//移動ThreadPool的ptr
func (m *MultipleScheduler) next() {
	m.ptr = (m.ptr + 1) % m.poolsize
}
func (m *MultipleScheduler) String() string {
	return fmt.Sprintf(
		"Scheduler:"+
			"\n\t|-Type:%T \n\t|-ThreadPoolSize:%d \n\t|-ChanBuf: %d\n\t|-%v",
		m, m.poolsize, cap(m.threadPool[0]), m.requestStorage,
	)
}
