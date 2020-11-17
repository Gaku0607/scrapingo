package logger

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

type Logger struct {
	Config *LoggerConfig
	mu     *sync.Mutex
}

//scrapingo默認使用的Logger格式
func DefaultLogger() *Logger {
	l := &Logger{}
	l.Config = DefaultLoggerConfig()
	l.mu = &sync.Mutex{}
	return l
}

//輸出DebugPrint
func (l *Logger) DebugPrint(val string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !strings.HasSuffix(val, "\n") {
		val += "\n"
	}
	fmt.Fprint(l.Config.Out, val)
}

//輸出Log
func (l *Logger) Log(Parms *LoggerFormatterParms) {
	l.DebugPrint(l.Config.LogFormat(Parms))
}

//關閉Logger的相關資源
//系統結束時記得調用Engine的Close OR Collector的Close 避免外洩
func (l *Logger) Close() {
	l.Config.close()
}

//Out為Log的輸出位置 Format為Log輸出時的格式
//可自定義LoggerConfig更改系統默認的LoggerConfig格式
type LoggerConfig struct {
	Format LoggerFormatter
	Out    io.Writer
}

//scrapingo默認使用的LoggerConfig格式
func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Format: defaultFormatter,
		Out:    defaultWrite,
	}
}

//調用defaultJSONFormatter 初始化LoggerConfig
func JSONLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Format: defaultJSONFormatter,
		Out:    defaultWrite,
	}
}

//傳入自定義的LoggerFormatter 初始化LoggerConfig
func LoggerConfigWithFormat(f LoggerFormatter) *LoggerConfig {
	return NewLoggerConfig(nil, f)
}

//傳入指定的io.Writer 初始化LoggerConfig
func LoggerConfigWithWrite(w io.Writer) *LoggerConfig {
	return NewLoggerConfig(w, nil)
}

func NewLoggerConfig(out io.Writer, Format LoggerFormatter) *LoggerConfig {
	conf := &LoggerConfig{}
	conf.Out = out
	conf.Format = Format

	if conf.Out == nil {
		conf.Out = defaultWrite
	}
	if conf.Format == nil {
		conf.Format = Format
	}

	return conf
}
func (l *LoggerConfig) LogFormat(Parms *LoggerFormatterParms) string {
	return l.Format(Parms)
}

//當Out為WriteCloser 轉換類型方便關閉
func (l *LoggerConfig) close() {
	if wc, ok := l.Out.(io.WriteCloser); ok {
		wc.Close()
	}
}
