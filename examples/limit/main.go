package main

import (
	"time"

	"github.com/Gaku0607/scrapingo"
)

func main() {
	//限制器
	limiter := &scrapingo.Limiter{
		Parallelcount:   10,                             //最大同時參訪數量
		DomainGlob:      "https://www.google.com/*",     //作用的URL 請參考glob.Glob的匹配條件
		DelayTime:       time.Duration(2) * time.Second, //延遲時間
		RandomDelayTime: time.Duration(1) * time.Second, //隨機延遲時間
	}
	//傳入協程數
	engine := scrapingo.NewEngine(10)
	engine.AddLimit(limiter)

	req, err := scrapingo.NewRequest("https://www.google.com/?hl=zh_tw")
	if err != nil {
		panic(err.Error())
	}

	engine.Run(req)

	engine.Wait()
	//關閉Logger,Persist等資源
	engine.Close()
}
