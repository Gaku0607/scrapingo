package main

import (
	"context"
	"time"

	"github.com/Gaku0607/scrapingo"
	"github.com/Gaku0607/scrapingo/persist"
	"github.com/gomodule/redigo/redis"
)

func main() {
	//scrapingo支持Gorm所支持的SQL 以及 Redis, JSONfile 儲存
	//實現scrapingo.Persist interface 即可自定義
	file, err := jsonfile()
	if err != nil {
		panic(err.Error())
	}
	engine := scrapingo.NewEngine(
		10,                            //協程數
		scrapingo.EnginePersist(file), //engine默認不支持任何儲存必須自行添加
		scrapingo.SchedulerStorage(&scrapingo.InMemoryRequestQueue{MaxSize: 500000}), //設置調度器儲存Request的最大容量
	)

	req, err := scrapingo.NewRequest("https://www.google.com")
	if err != nil {
		panic(err.Error())
	}
	//提交Request
	engine.Submit(req)
	//設置engine的作用時間
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(3)*time.Second)
	defer func() {
		//關閉Logger以及Persist資源
		engine.Close()
		cancel()
	}()
	//啟動engine
	engine.RunWithContext(ctx)
	//直到爬取結束為止進行等待
	engine.Wait()
}
func jsonfile() (persist.Persist, error) {
	return persist.NewPersistStore(
		persist.JSONFILE,
		persist.FileOptions(
			"./scrapingo.rtf",
			persist.SplitSize(1024*1024*10),
		),
	)
}

type Gorm struct {
	scrapingo.Model
}

func gorm() (persist.Persist, error) {
	return persist.NewPersistStore(
		persist.SQL,
		persist.SQLOptions(
			"mysql",
			"GormContent.....",
			persist.GormModel(&Gorm{}),
			persist.MaxIdleConns(100),
			persist.MaxOpenConns(200),
			persist.MaxConnLifeTime(10*time.Second),
		))
}
func redigo() (persist.Persist, error) {
	return persist.NewPersistStore(
		persist.REDIS,
		persist.RedisOptions(
			"127.0.0.1:6379",
			persist.DialOption(
				redis.DialPassword("12345"),
			),
			persist.MaxIdleConns(10),
			persist.MaxOpenConns(20),
			persist.MaxConnLifeTime(10*time.Second),
			persist.Wait(true)))
}
