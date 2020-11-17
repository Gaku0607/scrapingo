package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/Gaku0607/scrapingo"
	"github.com/Gaku0607/scrapingo/logger"
	"github.com/Gaku0607/scrapingo/persist"
)

var OutPut = os.Stdout

func OutPrint(b *Baha) error {
	data, err := json.Marshal(b)
	data = append(data, []byte("\n")...)
	if err != nil {
		return err
	}
	_, err = OutPut.Write(data)
	return err
}

type Baha struct {
	scrapingo.Model
	Title   string
	Time    time.Time
	Like    int
	Bad     int
	Author  string
	Account string
}

var DEC = "Gormcontent...."

func gorm() (persist.Persist, error) {
	return persist.NewPersistStore(
		persist.SQL,
		persist.SQLOptions(
			"mysql",
			DEC,
			persist.GormModel(&Baha{}),
			persist.MaxIdleConns(10),
			persist.MaxOpenConns(20),
			persist.MaxConnLifeTime(5*time.Second),
		))
}
func main() {
	out, err := os.OpenFile("./baha.rtf", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0664)

	if err != nil {
		panic(err.Error())
	}
	OutPut = out

	limiter := &scrapingo.Limiter{
		DelayTime:       time.Duration(1) * time.Second,
		RandomDelayTime: time.Duration(1500) * time.Millisecond,
		Parallelcount:   12,
		DomainGlob:      "https://forum.gamer.com.tw/*",
	}
	c := scrapingo.NewCollector(
		scrapingo.UserAgent("Your User-Agent"),
		scrapingo.RequestTimeOut(time.Duration(3)*time.Second),
		scrapingo.LoggerMode(true),
		scrapingo.ResultLogKey(
			func(pr *scrapingo.ParseResult) logger.LogKey {
				for _, item := range pr.Items {
					if Baha, ok := item.(*Baha); ok && Baha.Like == 999 {
						if err = OutPrint(Baha); err != nil {
							log.Println(err)
						}
					}
				}
				return scrapingo.DefaultResultLogKey(pr)
			},
		),
	)

	g, err := gorm()
	if err != nil {
		panic(err.Error())
	}

	e := scrapingo.NewEngine(
		20,
		scrapingo.EngineCollector(c),
		scrapingo.EnginePersist(g),
		scrapingo.SchedulerPool(20, 100),
	)

	defer e.Close()

	if err := e.AddLimit(limiter); err != nil {
		panic(err.Error())
	}

	var reqs []*scrapingo.Request

	req, err := scrapingo.NewRequest(
		"https://forum.gamer.com.tw/B.php?page=2999&bsn=60076",
		scrapingo.ParseFunction(ArticleList),
	)
	if err != nil {
		panic(err.Error())
	}

	reqs = append(reqs, req)

	for i := 2; i <= 100; i++ {
		if req, err = req.New(fmt.Sprintf("https://forum.gamer.com.tw/B.php?page=%d&bsn=60076", i)); err != nil {
			continue
		}
		reqs = append(reqs, req)
	}

	e.Run(reqs...)
	e.Wait()
}

var (
	b_list = regexp.MustCompile(`<table class="b-list">([\s\S]+?)</table>`)
	title  = regexp.MustCompile(`<p data-gtm="B頁文章列表-縮圖" href="([\s\S]+?)" class="b-list__main__title">([^<]+?)<`)

	author   = regexp.MustCompile(`class="username" target="_blank">([^<]+?)</a>`)
	account  = regexp.MustCompile(`class="userid" target="_blank">([^<]+?)</a>`)
	like     = regexp.MustCompile(`<span class="postgp">推<span>(.+?)</span></span>`)
	bad      = regexp.MustCompile(`<span class="postbp">噓<span>(.+?)</span></span>`)
	posttime = regexp.MustCompile(`class="edittime [\s\S]+? data-mtime="([^"]+?)"`)
)

func ArticleList(body []byte) (result *scrapingo.ParseResult) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			result = new(scrapingo.ParseResult)
		}
	}()
	list := b_list.Find(body)
	matches := title.FindAllSubmatch(list, -1)

	result = new(scrapingo.ParseResult)
	for _, match := range matches {
		BaHa := new(Baha)
		BaHa.Title = string(match[2])
		req, err := scrapingo.NewRequest(
			string(match[1]),
			scrapingo.ParseFunction(
				func(b []byte) *scrapingo.ParseResult {
					return Article(b, BaHa)
				},
			),
		)
		if err != nil {
			continue
		}
		result.Requests = append(result.Requests, req)
	}
	return result
}
func Article(body []byte, b *Baha) (result *scrapingo.ParseResult) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			result = new(scrapingo.ParseResult)
		}
	}()
	result = new(scrapingo.ParseResult)

	account := account.FindSubmatch(body)
	author := author.FindSubmatch(body)
	posttime := posttime.FindSubmatch(body)
	b.Account = string(account[1])
	b.Author = string(author[1])
	b.Time, _ = time.Parse("2006-01-02 15:04:05", string(posttime[1]))

	b.Bad, _ = strconv.Atoi(string(bad.FindSubmatch(body)[1]))

	like := like.FindSubmatch(body)

	var likecount int
	var likestr string = string(like[1])

	likecount, _ = strconv.Atoi(likestr)

	if likestr == "爆" || likestr == "x" {
		likecount = 999
	}

	b.Like = likecount
	result.Items = append(result.Items, b)

	return result
}
