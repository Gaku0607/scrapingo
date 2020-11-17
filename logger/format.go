package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

//crapingo默認LogTimeFormat
var defaultTimeFormat = time.Now().Format("2006/01/02 - 15:04:05")

//獲取當前的TimeFormat
func GetTimeFormat() string {
	return defaultTimeFormat
}

//修改當前的TimeFormat
func SetTimeFormat(timeFormat string) {
	defaultTimeFormat = timeFormat
}

const (
	green = "\033[97;42m"
	white = "\033[90;47m"
	red   = "\033[97;41m"
	blue  = "\033[97;44m"
	reset = "\033[0m"
)

//默認終端機輸出Log時會帶有顏色
//當Log輸出不為終端機時 建議關閉
var isOutColor = true

//獲取當前的是否輸出顏色
func IsOutColor() bool {
	return isOutColor
}

//修改當前Log輸出時是否帶顏色
func SetOutColor(b bool) {
	isOutColor = b
}

//Log輸出時的格式  可以參考 defaultFormatter 的實現方式
type LoggerFormatter func(*LoggerFormatterParms) string

//系統默認的輸出位置
//使用自定義的LoggerConfig即可指定輸出位置
var defaultWrite = os.Stdout

var (
	//scrapingo默認的輸出格式 實現LoggerFormatter即可自定義
	//使用自定義的LoggerConfig即可指定輸出位置
	defaultFormatter = func(f *LoggerFormatterParms) string {
		var typecolor string
		if IsOutColor() {
			typecolor = f.setTypeColor()
		}
		str := fmt.Sprintf("[SCRAPINGO] %s |%s %s %s| RequestID:%d | %s | URL:%s |",
			f.Time,
			typecolor, f.Type, reset,
			f.RequsetId, f.Method, f.URL,
		)
		for key, val := range f.Key {
			str = strings.Join([]string{str, fmt.Sprintf(" %s: %v |", key, val)}, "")
		}
		return str
	}

	defaultJSONFormatter = func(f *LoggerFormatterParms) string {
		s, _ := json.Marshal(f)
		return string(s)
	}
)

//自定義的Log輸出參數
type LogKey map[string]interface{}

//Logger.Log()輸出時所帶有的參數 key參數能夠自定義
type LoggerFormatterParms struct {
	Type      string `json:"type"`
	Time      string `json:"time"`
	RequsetId int64  `json:"requestId"`
	URL       string `json:"url"`
	Method    string `json:"method"`
	Key       LogKey `json:"key"`
}

func (l *LoggerFormatterParms) setTypeColor() string {
	switch l.Type {
	case " ERROR ":
		return red
	case "REQUEST":
		return white
	case "RESULT ":
		return green
	default:
		return blue
	}
}

func CreateLogParms(reqId int64, Type, url, method string, key map[string]interface{}) *LoggerFormatterParms {
	l := &LoggerFormatterParms{}
	l.Time = GetTimeFormat()
	l.RequsetId = reqId
	l.Type = Type
	l.URL = url
	l.Method = method
	l.Key = key
	return l
}
