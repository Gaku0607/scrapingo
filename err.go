package scrapingo

import "errors"

var (
	//scrapingo.Engine的closed參數為true 仍調用Run or RunWithContext 時的錯誤
	ErrEngineIsClosed = errors.New("scrapingo: Persitst is closed")
	//調用scrapingo.Engine.RunWithContext()傳入nil時的錯誤
	ErrContextIsNil = errors.New("scrapingo: context is nil")
	//調用Collector的checkRequestInfo()時 scrapingo.Request中的URL.String() == "" 時的錯誤
	ErrURLMiss = errors.New("scrapingo: URL Missing")
	//當Request的儲存總數超過所設定的值時進行Panic
	ErrOverMaxRequestStorage = errors.New("scrapingo: RequestStorage MaxSize Reached")
	//當limiter的參數urlGlob為match.Nothing時的錯誤
	ErrlimiterNoParttern = errors.New("scrapingo: limiter cannt No Parttern")
	//當重複訪問相同URL時發生此錯誤
	ErrIsVisitedURL = errors.New("scrapingo: URL is Visited")
)
