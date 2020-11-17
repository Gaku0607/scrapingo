package scrapingo

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Gaku0607/scrapingo/logger"
)

//Collector的可選參數
type CollectorOption func(*Collector)

//修改Collector的默認LoggerMode
func LoggerMode(b bool) CollectorOption {
	return func(c *Collector) {
		c.LoggerMode = b
	}
}

//修改Collector的默認LoggerConfig
func LoggerConfig(l *logger.LoggerConfig) CollectorOption {
	return func(c *Collector) {
		c.logger.Config = l
	}
}

//修改Collector的默認RequestLogKey
func RequestLogKey(f func(*Request) logger.LogKey) CollectorOption {
	return func(c *Collector) {
		c.requestlogkey = f
	}
}

//修改Collector的默認ErrLogKey
func ErrLogKey(f func(*Request, error) logger.LogKey) CollectorOption {
	return func(c *Collector) {
		c.errlogkey = f
	}
}

//修改Collector的默認ResultLogKey
func ResultLogKey(f func(*ParseResult) logger.LogKey) CollectorOption {
	return func(c *Collector) {
		c.resultlogkey = f
	}
}

var (
	//Collector默認所使用的ErrLogKey
	DefaultErrLogKey = func(_ *Request, e error) logger.LogKey {
		return logger.LogKey{"errMsg": e.Error()}
	}
	//Collector默認所使用的ReqLogKey
	DefaultReqLogKey = func(*Request) logger.LogKey {
		return logger.LogKey{}
	}
	//Collector默認所使用的ResultLogKey
	DefaultResultLogKey = func(p *ParseResult) logger.LogKey {
		return logger.LogKey{
			"requestCount": len(p.Requests),
			"itmesCount":   len(p.Items),
		}
	}
)

//修改Collector的默認
func MaxBodySize(b int) CollectorOption {
	return func(c *Collector) {
		c.MaxBodySize = b
	}
}

//修改Collector的默認
func MaxDepth(d int) CollectorOption {
	return func(c *Collector) {
		c.MaxDepth = d
	}
}

//修改Collector的默認UserAgent
func UserAgent(s string) CollectorOption {
	return func(c *Collector) {
		c.UserAgent = s
	}
}

//修改Collector的默認最大請求時間
func RequestTimeOut(t time.Duration) CollectorOption {
	return func(c *Collector) {
		c.transfer.Client.Timeout = t
	}
}

//修改Collector的默認的URL去重儲存
func VisitedStorage(v VisitStorage) CollectorOption {
	return func(c *Collector) {
		c.visitedStorage = v
	}
}

type Collector struct {

	//當進行請求時Request若沒設置UserAgent則會使用Collector的UserAgent

	UserAgent string

	//當爬蟲到所指定的最大深度時返回不進行爬取
	//MaxDepth為0時則沒有上限

	MaxDepth int

	//responsBody最大的Size當超出指定大小時 將進行裁減至指定的尺寸
	//MaxBodySize為零時則沒有上限

	MaxBodySize int

	//當爬取URL或是Engine的Saveitem函數發生錯誤時會調用自定義的ErrCallback函數
	//調用OnErr即可自行添加

	errcallbacks []ErrCallbackContainer

	//當調用checkRequestInfo()後會調用自定義的RequestCallback函數
	//調用OnRequest即可自行添加

	requestcallbacks []RequestCallbackContainer

	//當爬取完成時會調用自定義的ResultCallback函數
	//調用OnResult即可自行添加

	resultcallbacks []ResultCallbackContainer

	//LoggerMode為true時會輸出Log
	//Collector默認開啟Logger

	LoggerMode bool

	//Collector默認使用 logger.DefaultLogger()
	//調用CollectorOption LoggerConfig 可自行修改輸出格式
	//(參考logger.Logger logger.LoggerConfig)

	logger *logger.Logger

	//Log輸出REQUEST時 可自定義RequestLogKey的部分
	//Collector默認使用defaultRequestLogKey

	requestlogkey func(*Request) logger.LogKey

	//Log輸出ERROR時 可自定義ErrLogKey的部分
	//Collector默認使用defaultErrLogKey

	errlogkey func(*Request, error) logger.LogKey

	//Log輸出Result時 可自定義ResultLogKey的部分
	//Collector默認使用defaultResultLogKey

	resultlogkey func(*ParseResult) logger.LogKey

	//返回的item總數 使用於每個Item的唯一識別

	itemcount int64

	//Request的總數 使用於每個Request的唯一識別

	requestcount int64

	//儲存訪問過的URL防止重複訪問
	//可以參考(scrapingo.visitStorage)interface自行定義

	visitedStorage VisitStorage

	transfer *Transfer
	mu       *sync.Mutex
	ctx      context.Context
}

//當爬取URL或是Engine的Saveitem函數發生錯誤時會調用
type ErrCallback func(*Request, error)

//當調用checkRequestInfo()後會調用自定義的RequestCallback函數
type RequestCallback func(*Request)

//當爬取完成時會調用該函數
type ResultCallback func(*ParseResult)

//Id為自行定義的唯一識別調用 調用OnErrDetach刪除時所使用
type ErrCallbackContainer struct {
	Id   int
	Func ErrCallback
}

//Id為自行定義的唯一識別調用 調用OnRequestDetach刪除時所使用
type RequestCallbackContainer struct {
	Id   int
	Func RequestCallback
}

//Id為自行定義的唯一識別調用 調用OnResultDetach刪除時所使用
type ResultCallbackContainer struct {
	Id   int
	Func ResultCallback
}

//使用NewCollector()初始化時會調用 DefaultParms()
//傳入CollectorOption即可覆蓋默認值
func NewCollector(options ...CollectorOption) *Collector {
	c := &Collector{}
	c.defualtParms()
	for _, option := range options {
		option(c)
	}
	return c
}

//Collector的默認參數
func (c *Collector) defualtParms() {
	c.UserAgent = ""
	c.transfer = &Transfer{}
	c.transfer.Client = http.Client{}
	c.LoggerMode = true
	c.logger = logger.DefaultLogger()
	c.visitedStorage = defaultHasStorage()
	c.requestlogkey = DefaultReqLogKey
	c.errlogkey = DefaultErrLogKey
	c.resultlogkey = DefaultResultLogKey
	c.mu = &sync.Mutex{}
	c.ctx = context.Background()
}

//為每個Request設置唯一識別
func (c *Collector) setRequestId() int64 {
	return atomic.AddInt64(&c.requestcount, 1)
}

//為每個Item設置唯一識別
func (c *Collector) setItemId() int64 {
	return atomic.AddInt64(&c.itemcount, 1)
}

//傳入Request進行爬取
func (c *Collector) Request(req *Request) (*ParseResult, error) {
	return c.scraping(req.URL.String(), req.Header, req.Method, req.Depth+1, req.Body, context.Background(), req.Parse)
}

//傳入所需的參數進行爬取
func (c *Collector) Do(URL, Method string, Header http.Header, Body io.Reader, ctx context.Context, p ParseFunc) (*ParseResult, error) {
	return c.scraping(URL, Header, Method, 1, Body, ctx, p)
}

//傳入URL以及對應的解析（ParseFunc）進行爬取
func (c *Collector) Get(URL string, p ParseFunc) (*ParseResult, error) {
	return c.scraping(URL, http.Header{}, http.MethodGet, 1, nil, nil, p)
}

// Content-Type標頭設置为application / x-www-form-urlencoded。
//要設置其他自定義標頭，請使用Do or Request。
func (c *Collector) PostForm(URL string, data map[string]string, p ParseFunc) (*ParseResult, error) {
	return c.Post(URL, "application / x-www-form-urlencoded", createDataReader(data), p)
}

//要設置其他自定義標頭，請使用Do or Request。
func (c *Collector) Post(URL string, contentType string, Body io.Reader, p ParseFunc) (*ParseResult, error) {
	return c.scraping(URL, http.Header{"Content-Type": {contentType}}, http.MethodPost, 1, nil, nil, p)
}

//Id為刪除時的唯一標示 設置ErrCallback
//當爬取發生錯誤時會調用所設置的ErrCallback
func (c *Collector) OnErr(Id int, f ErrCallback) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errcallbacks = append(c.errcallbacks, ErrCallbackContainer{Id: Id, Func: f})
}

//Id為刪除時的唯一標示 設置RequestCallback
//當要進行爬取時會調用所設置的RequestCallback
func (c *Collector) OnRequest(Id int, f RequestCallback) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestcallbacks = append(c.requestcallbacks, RequestCallbackContainer{Id: Id, Func: f})
}

//Id為刪除時的唯一標示 設置ResultCallback
//當爬取結束時會調用所設置的ResultCallback
func (c *Collector) OnResult(Id int, f ResultCallback) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resultcallbacks = append(c.resultcallbacks, ResultCallbackContainer{Id: Id, Func: f})
}

//輸入指定Id會刪除對應的ErrCallback
func (c *Collector) OnErrDetach(Id int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for index, callback := range c.errcallbacks {
		if callback.Id == Id {
			c.errcallbacks = append(c.errcallbacks[:index], c.errcallbacks[index+1:]...)
		}
	}
}

//輸入指定Id會刪除對應的RequestCallback
func (c *Collector) OnRequestDetach(Id int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for index, callback := range c.requestcallbacks {
		if callback.Id == Id {
			c.requestcallbacks = append(c.requestcallbacks[:index], c.requestcallbacks[index+1:]...)
		}
	}
}

//輸入指定Id會刪除對應的ResultCallback
func (c *Collector) OnResultDetach(Id int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for index, callback := range c.resultcallbacks {
		if callback.Id == Id {
			c.resultcallbacks = append(c.resultcallbacks[:index], c.resultcallbacks[index+1:]...)
		}
	}
}

//將會調用自定義的ErrCallback
//當LoggerMode為true時會調用指定的Logger
func (c *Collector) handleOnErr(r *Request, err error) {
	if c.LoggerMode {
		Parms := logger.CreateLogParms(r.ID, " ERROR ", r.URL.String(), r.Method, c.errlogkey(r, err))
		c.logger.Log(Parms)
	}
	for _, callback := range c.errcallbacks {
		callback.Func(r, err)
	}
}

//將會調用自定義的RequestCallback
//當LoggerMode為true時會調用指定的Logger
func (c *Collector) handleOnRequest(r *Request) {
	if c.LoggerMode {
		Parms := logger.CreateLogParms(r.ID, "REQUEST", r.URL.String(), r.Method, c.requestlogkey(r))
		c.logger.Log(Parms)
	}
	for _, callback := range c.requestcallbacks {
		callback.Func(r)
	}
}

//將會調用自定義的ParseResultCallback
//當LoggerMode為true時會調用指定的Logger
func (c *Collector) handleOnResult(r *ParseResult) {
	if c.LoggerMode {
		Parms := logger.CreateLogParms(r.ParentRequest.ID, "RESULT ", r.ParentRequest.URL.String(),
			r.ParentRequest.Method, c.resultlogkey(r))
		c.logger.Log(Parms)
	}
	for _, callback := range c.resultcallbacks {
		callback.Func(r)
	}
}

//爬取所指定的URL 並使用傳入的ParseFunc進行相對應的解析
//會調用所指定的Callback函數
func (c *Collector) scraping(u string, Header http.Header, Method string, Depth int, Body io.Reader, ctx context.Context, p ParseFunc) (*ParseResult, error) {
	req, err := c.checkRequsetInfo(u, Header, Method, Body, Depth, ctx, p)

	if err != nil {
		return nil, err
	}

	c.handleOnRequest(req)

	rc, ok := Body.(io.ReadCloser)
	if !ok && Body != nil {
		rc = ioutil.NopCloser(Body)
	}

	httpReq := &http.Request{
		Method:     req.Method,
		URL:        req.URL,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     req.Header,
		Body:       rc,
		Host:       removeEmptyPort(req.URL.Host),
	}

	setRequsetBody(httpReq, Body)
	httpReq = httpReq.WithContext(ctx)

	respbody, err := c.transfer.do(httpReq, c.MaxBodySize)
	if err != nil {
		c.handleOnErr(req, err)
		return nil, err
	}

	ParseResult := req.Parse(respbody)
	ParseResult.ParentRequest = req

	c.setResultInfo(ParseResult)

	c.handleOnResult(ParseResult)

	return ParseResult, nil
}

//當item有中有設置scrapingo.Model 或者是 相關參數時
//將賦予參數所對應的值  （請查看scrapingo.Model的說明）
func (c *Collector) setResultInfo(result *ParseResult) {
	for _, req := range result.Requests {
		req.URL, _ = result.ParentRequest.URL.Parse(req.URL.String())
	}
	for _, item := range result.Items {
		itemID := c.setItemId()
		if reflect.Ptr != reflect.TypeOf(item).Kind() {
			continue
		}
		if reflect.Struct != reflect.TypeOf(item).Elem().Kind() {
			continue
		}
		f := reflect.ValueOf(item).Elem()
		var val reflect.Value

		for i, n := 0, f.NumField(); i < n; i++ {
			val = f.Field(i)
			if val.Type() == reflect.TypeOf(Model{}) {
				val.Field(0).SetInt(itemID)
				val.Field(1).SetInt(result.ParentRequest.ID)
				val.Field(2).SetString(result.ParentRequest.URL.String())
			}
			if val.Type().Name() == "ItemID" && val.Type().Kind() == reflect.Int {
				val.SetInt(itemID)
			}
			if val.Type().Name() == "ParentID" && val.Type().Kind() == reflect.Int {
				val.SetInt(result.ParentRequest.ID)
			}
			if val.Type().Name() == "ParentURL" && val.Type().Kind() == reflect.String {
				val.SetString(result.ParentRequest.URL.String())
			}
		}
	}
}

//確認請求內容 當產生錯誤時將不進行請求
//也不會調用Callback函數
func (c *Collector) checkRequsetInfo(u string, Header http.Header, Method string, Body io.Reader, Depth int, ctx context.Context, p ParseFunc) (*Request, error) {
	URL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	if u == "" {
		return nil, ErrURLMiss
	}

	if c.MaxDepth > 0 && Depth > c.MaxDepth {
		return nil, fmt.Errorf("scrapingo: RequestDepth is %d ,Over MaxDepth %d", Depth, c.MaxDepth)
	}
	if ctx == nil {
		ctx = c.ctx
	}
	if Method == "" {
		Method = http.MethodGet
	}

	var hascode uint64
	f := fnv.New64a()

	f.Write([]byte(URL.String()))

	if Method == http.MethodGet {
		hascode = f.Sum64()
	} else if Body != nil {
		bytes := readertobyte(Body)
		f.Write(bytes)
		hascode = f.Sum64()
	}

	if c.isVisitd(hascode) {
		return nil, ErrIsVisitedURL
	}
	c.Visited(hascode)

	if Header.Get("User-Agent") == "" {
		Header.Add("User-Agent", c.UserAgent)
	}
	if Method == http.MethodPost && Header.Get("Content-Type") == "" {
		Header.Add("Content-Type", "application / x-www-form-urlencoded")
	}
	if p == nil {
		p = NilParse
	}
	return &Request{
		ID:     c.setRequestId(),
		URL:    URL,
		Ctx:    ctx,
		Header: Header,
		Method: Method,
		Body:   Body,
		Depth:  Depth,
		Parse:  p,
	}, nil
}
func removeEmptyPort(host string) string {
	if strings.LastIndex(host, ":") > strings.LastIndex(host, "]") {
		return strings.TrimSuffix(host, ":")
	}
	return host
}
func setRequsetBody(req *http.Request, Body io.Reader) {
	if Body != nil {
		switch v := Body.(type) {
		case *bytes.Buffer:
			req.ContentLength = int64(v.Len())
			buf := v.Bytes()
			req.GetBody = func() (io.ReadCloser, error) {
				r := bytes.NewReader(buf)
				return ioutil.NopCloser(r), nil
			}
		case *bytes.Reader:
			req.ContentLength = int64(v.Len())
			snapshot := *v
			req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return ioutil.NopCloser(&r), nil
			}
		case *strings.Reader:
			req.ContentLength = int64(v.Len())
			snapshot := *v
			req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return ioutil.NopCloser(&r), nil
			}
		}
		if req.GetBody != nil && req.ContentLength == 0 {
			req.Body = http.NoBody
			req.GetBody = func() (io.ReadCloser, error) { return http.NoBody, nil }
		}
	}
}

//確認是否有重複訪問
func (c *Collector) isVisitd(reqId uint64) bool {
	return c.visitedStorage.IsVisited(reqId)
}

//將訪問過的URL轉為哈希值儲存
func (c *Collector) Visited(reqId uint64) {
	c.visitedStorage.Visited(reqId)
}

//將io.Reader轉為[]byte
func readertobyte(reader io.Reader) []byte {
	buf := &bytes.Buffer{}
	buf.ReadFrom(reader)
	if strreader, ok := reader.(*strings.Reader); ok {
		strreader.Seek(0, 0)
	} else if bytesreader, ok := reader.(*bytes.Reader); ok {
		bytesreader.Seek(0, 0)
	}
	return buf.Bytes()
}

//創建PostForm所需要將map[string]string轉為io.Reader
func createDataReader(data map[string]string) io.Reader {
	val := url.Values{}
	for k, v := range data {
		val.Add(k, v)
	}
	return strings.NewReader(val.Encode())
}

//添加對請求時對URL的限制
func (c *Collector) AddLimit(l *Limiter) error {
	return c.transfer.AddLimiter(l)
}

//添加對請求時對URL的限制
func (c *Collector) AddLimits(l []*Limiter) error {
	return c.transfer.AddLimiters(l)
}

//關閉Logger資源
func (c *Collector) Close() {
	c.logger.Close()
}

//Clone後的callback函式需重新定義
func (c *Collector) Clone() *Collector {
	return &Collector{
		UserAgent:        c.UserAgent,
		MaxDepth:         c.MaxDepth,
		MaxBodySize:      c.MaxBodySize,
		requestcount:     c.requestcount,
		itemcount:        c.itemcount,
		mu:               c.mu,
		transfer:         c.transfer,
		logger:           c.logger,
		LoggerMode:       c.LoggerMode,
		errcallbacks:     make([]ErrCallbackContainer, 0),
		requestcallbacks: make([]RequestCallbackContainer, 0),
		resultcallbacks:  make([]ResultCallbackContainer, 0),
		errlogkey:        c.errlogkey,
		requestlogkey:    c.requestlogkey,
		resultlogkey:     c.resultlogkey,
	}
}

func (c *Collector) String() string {
	return fmt.Sprintf(
		"Collector:"+
			"\n\t|-User-Agent:%s\n\t|-LoggerMode:%v\n\t|-MaxDepth:%d\n\t|-MaxBodySize:%d\n\t|-requestcount:%d\n\t|-itemcount:%d\n\t|-"+
			"errcallbackcount:%d\n\t|-requestcallbackcount:%d\n\t|-resultcallbackcount:%d\n\t|-"+
			"%s\n\t\t",
		c.UserAgent, c.LoggerMode, c.MaxDepth, c.MaxBodySize, c.requestcount, c.itemcount,
		len(c.errcallbacks), len(c.requestcallbacks), len(c.resultcallbacks),
		c.transfer,
	)
}
