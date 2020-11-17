package scrapingo

import (
	"context"
	"fmt"
	"sync"

	"github.com/Gaku0607/scrapingo/persist"
)

//Engine的可選參數
type EngineOption func(*ConcurrentEngine)

//修改默認引擎的調度器須將調度器其他參數給傳入
//ThreadPool大小最大為引擎的ThreadCount
func EngineScheduler(s Scheduler, p int, bufsize int, q RequestStorage) EngineOption {
	return func(e *ConcurrentEngine) {
		if p > e.ThreadCount {
			p = e.ThreadCount
		}
		e.engineScheduler = s
		e.engineScheduler.ConfigPool(p, bufsize)
		e.engineScheduler.ConfigStorage(q)
	}
}

//修改默認引擎的調度器的ThreadPool大小
//ThreadPool大小最大將不超過引擎的ThreadCount
func SchedulerPool(p int, bufsize int) EngineOption {
	return func(e *ConcurrentEngine) {
		if p > e.ThreadCount {
			p = e.ThreadCount
		}
		e.engineScheduler.ConfigPool(p, bufsize)
	}
}

//修改默認引擎的調度器的RequestStorage
func SchedulerStorage(s RequestStorage) EngineOption {
	return func(e *ConcurrentEngine) {
		e.engineScheduler.ConfigStorage(s)
	}
}

//修改默認引擎的儲存格式
func EnginePersist(p persist.Persist) EngineOption {
	return func(e *ConcurrentEngine) {
		e.persist = p
	}
}

//修改默認引擎的Collector
func EngineCollector(c *Collector) EngineOption {
	return func(e *ConcurrentEngine) {
		e.C = c
	}
}

type ConcurrentEngine struct {

	//Engine默認使用MultipleScheduler
	//參考（scrapingo.scheduler）interface的實現方式

	engineScheduler Scheduler

	//儲存格式 Engine默認不支持任何儲存格式
	//NewEngine時調用EngineOption Persist進行修改
	//參考（persist.Persist）interface的實現方式

	persist persist.Persist

	ThreadCount int //開啟協程數

	//當調用了Close()關閉資源會設為true
	//在調用Run or RunWithContext時會出現錯誤

	closed bool

	C  *Collector
	mu sync.Mutex
	wg *sync.WaitGroup
}

//傳入ThreadCount初始化Engine
//會調用defaultParms()獲取默認參數
//傳入EngineOption即可覆蓋默認值
func NewEngine(ThreadCount int, options ...EngineOption) *ConcurrentEngine {
	e := &ConcurrentEngine{ThreadCount: ThreadCount}
	e.defaultParms()
	for _, option := range options {
		option(e)
	}
	return e
}

//引擎的默認值
func (e *ConcurrentEngine) defaultParms() {
	e.engineScheduler = &MultipleScheduler{}
	e.engineScheduler.ConfigPool(e.ThreadCount, 0)
	e.engineScheduler.ConfigStorage(DefaultStorage())
	e.C = NewCollector()
	e.persist = &persist.NilPersist{}
	e.wg = &sync.WaitGroup{}
}

//調用Run or RunContext時
//必須調用該函數進行等待
func (e *ConcurrentEngine) Wait() {
	e.wg.Wait()
}

//啟動引擎 調用時必須傳入種子
//或者在調用該函數前調用Subimt傳入Request
//否則直接結束
func (e *ConcurrentEngine) Run(seeds ...*Request) error {
	return e.RunWithContext(context.Background(), seeds...)
}

//啟動引擎 調用時必須傳入種子
//或者在調用該函數前調用Subimt傳入Request
//否則直接結束
//傳入Context 可自行Cancel 結束引擎
func (e *ConcurrentEngine) RunWithContext(ctx context.Context, seeds ...*Request) (err error) {
	if ctx == nil {
		return ErrContextIsNil
	}
	if e.closed {
		return ErrEngineIsClosed
	}
	var complete chan struct{} = make(chan struct{})

	for i := 0; e.ThreadCount > i; i++ {
		e.wg.Add(1)
		e.scheduler(e.engineScheduler.RequestChan(), complete)
	}
	for _, s := range seeds {
		e.engineScheduler.Submit(s)
	}
	e.engineScheduler.Run(ctx, complete)
	return
}

//接收Sheduler傳來的Request進行爬取以及儲存結果
func (e *ConcurrentEngine) scheduler(in <-chan *Request, complete chan<- struct{}) {
	go func() {
		defer e.wg.Done()
		for req := range in {
			ParseResult, err := e.C.Request(req)
			if err != nil {
				complete <- struct{}{}
				continue
			}
			for _, item := range ParseResult.Items {
				if err = e.itemSave(item); err != nil {
					e.C.handleOnErr(ParseResult.ParentRequest, err)
				}
			}
			for _, request := range ParseResult.Requests {
				request.Depth = req.Depth
				e.engineScheduler.Submit(request)
			}
			complete <- struct{}{}
		}
	}()
}

//儲存item 默認不支持任何儲存
//NewEngine時調用EngineOption Persist進行修改
func (e *ConcurrentEngine) itemSave(item interface{}) error {
	return e.persist.Save(item)
}

//添加對請求時對URL的限制
func (e *ConcurrentEngine) AddLimit(l *Limiter) error {
	return e.C.AddLimit(l)
}

//添加對請求時對URL的限制
func (e *ConcurrentEngine) AddLimits(l []*Limiter) error {
	return e.C.AddLimits(l)
}

//提交新的Request至Scheduler
func (e *ConcurrentEngine) Submit(req *Request) {
	e.engineScheduler.Submit(req)
}
func (e *ConcurrentEngine) Submits(reqs []*Request) {
	for _, req := range reqs {
		e.Submit(req)
	}
}

//結束前必須關閉持久化以及Logger資源
func (e *ConcurrentEngine) Close() {
	e.persist.Close()
	e.C.Close()
	e.closed = true
}

func (e *ConcurrentEngine) Clone() *ConcurrentEngine {
	return &ConcurrentEngine{
		engineScheduler: e.engineScheduler,
		ThreadCount:     e.ThreadCount,
		persist:         e.persist,
		C:               e.C,
		wg:              &sync.WaitGroup{},
		closed:          e.closed,
	}
}

func (e *ConcurrentEngine) String() string {
	return fmt.Sprintf(
		"Engine:\n|-"+
			"ThreadCount:%d \n|-enginePersist:%T \n|-%v\n|-%s ",
		e.ThreadCount, e.persist, e.engineScheduler, e.C,
	)
}
