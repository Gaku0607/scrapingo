package main

import (
	"fmt"
	"net/http"

	"github.com/Gaku0607/scrapingo"
	"github.com/Gaku0607/scrapingo/logger"
)

func main() {

	c := scrapingo.NewCollector(
		scrapingo.MaxBodySize(1024*500), //限制返回的responsBodySize
		scrapingo.MaxDepth(3),           //爬取最大深度
		scrapingo.RequestLogKey( //自定義RequestLogKey 輸出
			func(r *scrapingo.Request) logger.LogKey {
				if r.Method == http.MethodPost {
					return logger.LogKey{"Header": r.Header.Get("Content-Type")}
				}
				return logger.LogKey{}
			},
		),
	)
	c.OnErr( //當爬取發生error時會調用此函數
		1, //唯一識別ID     OnErrDetach(ID int)可以刪除ID對應的Callback函數
		func(req *scrapingo.Request, err error) {
			if req.ID > 10 {
				fmt.Println(err)
			}
		},
	)
	c.OnRequest( //當確認完Request內容發起請求時調用此函數
		1,
		func(req *scrapingo.Request) {
			if req.Depth > 3 {
				fmt.Println(req)
			}
		},
	)
	c.Get("https://www.google.com", scrapingo.NilParse)
}
