package scrapingo

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

//Request的可選參數
type RequestOption func(*Request)

//修改Request默認的method
func Method(m string) RequestOption {
	return func(r *Request) {
		r.Method = m
	}
}

//修改Request默認的Header
func Header(h http.Header) RequestOption {
	return func(r *Request) {
		r.Header = h
	}
}

//修改Request默認的Body
func Body(b io.Reader) RequestOption {
	return func(r *Request) {
		r.Body = b
	}
}

//修改Request默認的Context
func Ctx(ctx context.Context) RequestOption {
	return func(r *Request) {
		r.Ctx = ctx
	}
}

////修改Request默認的ParseFunc
func ParseFunction(f ParseFunc) RequestOption {
	return func(r *Request) {
		r.Parse = f
	}
}

//請求時所需要的URL 以及 URL所對應的解析函式
type Request struct {
	ID     int64 //Request的唯一識別
	URL    *url.URL
	Ctx    context.Context
	Header http.Header
	Depth  int
	Method string
	Body   io.Reader
	Parse  ParseFunc
}

func (r *Request) New(u string) (*Request, error) {
	URL, err := r.URL.Parse(u)
	if err != nil {
		return nil, err
	}
	return &Request{
		URL:    URL,
		Method: r.Method,
		Header: r.Header,
		Body:   r.Body,
		Parse:  r.Parse,
	}, nil
}
func NewRequest(u string, options ...RequestOption) (*Request, error) {
	URL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	req := &Request{}
	req.URL = URL
	req.Header = http.Header{}
	for _, option := range options {
		option(req)
	}
	return req, nil
}
