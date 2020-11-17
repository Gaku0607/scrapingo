package main

import (
	"fmt"
	"os"

	"github.com/Gaku0607/scrapingo"
	"github.com/Gaku0607/scrapingo/logger"
)

func main() {
	file, err := os.Open("./scrapingo.log")
	if err != nil {
		panic(err.Error())
	}
	//當Logger輸出為終端機以外時建議關閉顏色輸出
	logger.SetOutColor(false)

	conf := logger.LoggerConfigWithWrite(file)

	c := scrapingo.NewCollector(
		scrapingo.LoggerConfig(conf),
	)
	//關閉Logger資源
	defer c.Close()

	result, err := c.Get("https://www.google.com", scrapingo.NilParse)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(result)
}
