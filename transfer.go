package scrapingo

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gobwas/glob"
	"github.com/gobwas/glob/match"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

type Limiter struct {
	//請求延遲時間
	DelayTime time.Duration
	//隨機請求延遲時間
	RandomDelayTime time.Duration
	//最大平行數
	Parallelcount int
	//Chan的BufSize為Parallelcount
	waitChan chan struct{}
	//URL域名
	DomainGlob string
	//匹配URL域名
	urlGlob glob.Glob
}

//初始化limiter
func (l *Limiter) register() (err error) {
	l.urlGlob, err = glob.Compile(l.DomainGlob)
	if err != nil {
		return err
	}
	if _, ok := l.urlGlob.(match.Nothing); ok {
		return ErrlimiterNoParttern
	}
	size := l.Parallelcount
	if size <= 0 {
		size = 1
	}
	l.waitChan = make(chan struct{}, size)
	return nil
}
func (l *Limiter) Match(URL string) bool {
	return l.urlGlob.Match(URL)
}
func (l *Limiter) String() string {
	return fmt.Sprintf(
		"DelayTime:%.3fs RandomDelayTime:%.3fs Parallelcount:%d DomainGlob:%s",
		l.DelayTime.Seconds(), l.RandomDelayTime.Seconds(), l.Parallelcount, l.DomainGlob,
	)
}

type Transfer struct {
	Client   http.Client
	Limiters []*Limiter
	rw       sync.RWMutex
}

//取得註冊過的Limiter對指定的URL進行限制
func (t *Transfer) getLimiter(URL string) *Limiter {
	t.rw.RLock()
	defer t.rw.RUnlock()
	for _, limiter := range t.Limiters {
		if limiter.Match(URL) {
			return limiter
		}
	}
	return nil
}

//模擬請求返回解碼後的html[]Byte 當[]ByteSize大於傳入的MAxBodySize時進行限制
func (t *Transfer) do(req *http.Request, MaxBodySize int) ([]byte, error) {
	limiter := t.getLimiter(req.URL.String())

	if limiter != nil {
		limiter.waitChan <- struct{}{}
		defer func() {
			var randomDelayTime time.Duration
			if limiter.RandomDelayTime != 0 {
				randomDelayTime = time.Duration(rand.Int63n(int64(limiter.RandomDelayTime)))
			}

			time.Sleep(limiter.DelayTime + randomDelayTime)
			<-limiter.waitChan
		}()
	}

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scrapingo: Respons StatusCode is %d", resp.StatusCode)
	}
	return fetch(resp.Body, MaxBodySize)
}

//進行html解碼以及限制返回的ResponsBodySize
func fetch(Body io.Reader, MaxBodySize int) ([]byte, error) {
	BufReader := bufio.NewReader(Body)

	e, err := determinEncoding(BufReader)
	if err != nil {
		return nil, err
	}

	var BodyReader io.Reader
	BodyReader = BufReader

	if MaxBodySize > 0 && BufReader.Size() > MaxBodySize {
		BodyReader = io.LimitReader(BodyReader, int64(MaxBodySize))
	}

	NewReader := transform.NewReader(BodyReader, e.NewDecoder())
	return ioutil.ReadAll(NewReader)
}

//取1024byt探測html所使用的編碼方式
func determinEncoding(r *bufio.Reader) (encoding.Encoding, error) {
	bytes, err := r.Peek(1024)
	if err != nil {
		return nil, err
	}
	e, _, _ := charset.DetermineEncoding(bytes, "")
	return e, nil
}

//添加limiter至Transfer中當register()返回error時添加失敗
func (t *Transfer) AddLimiter(l *Limiter) (err error) {
	t.rw.Lock()
	defer t.rw.Unlock()
	if err = l.register(); err == nil {
		t.Limiters = append(t.Limiters, l)
	}
	return err
}

//添加limiter至Transfer中當register()返回error時添加失敗
func (t *Transfer) AddLimiters(limiters []*Limiter) (err error) {
	for _, l := range limiters {
		if err = t.AddLimiter(l); err != nil {
			return err
		}
	}
	return err
}
func (t *Transfer) String() string {
	str := fmt.Sprintf("Trandfer:\n\t\t|-RequestTimeOut: %.3fs", t.Client.Timeout.Seconds())
	for i, limiter := range t.Limiters {
		str = strings.Join([]string{str, fmt.Sprintf("|-limiter%d:", i+1), "|\t|-" + limiter.String()}, "\n\t\t")
	}
	return str
}
