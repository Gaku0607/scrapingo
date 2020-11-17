package persist

import (
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
)

//Persist的可選參數
type PersistOption func(*PersistOptions)

//文件最大容量 當超過時會進行切割文件
func SplitSize(c int64) PersistOption {
	return func(p *PersistOptions) {
		p.fileSize = c
	}
}

//開啟文件的模式
func FileMode(c os.FileMode) PersistOption {
	return func(p *PersistOptions) {
		p.fileMode = c
	}
}

//文件切割時 新文件的檔名的格式
func FileNameFormat(format string) PersistOption {
	return func(p *PersistOptions) {
		p.flieNameFormat = format
	}
}

//定義SqlModel
func GormModel(Models ...interface{}) PersistOption {
	return func(p *PersistOptions) {
		p.sqlModel = Models
	}
}

//最大閒置連結數 當小於等於0時最大閒置連結數為2
// 當最大連結數小於最大閒置連結數 則 最大閒置連結數會等於最大連結數
func MaxIdleConns(c int) PersistOption {
	return func(p *PersistOptions) {
		p.maxIdleConns = c
	}
}

//最大連結數 當最大連結數小餘等於0時 則 沒有上限
// 當最大連結數小於最大閒置連結數 則 最大閒置連結數會等於最大連結數
func MaxOpenConns(c int) PersistOption {
	return func(p *PersistOptions) {
		p.maxOpenConns = c
	}
}

//設置連結最大可復用時間
func MaxConnLifeTime(t time.Duration) PersistOption {
	return func(p *PersistOptions) {
		p.maxConnLifeTime = t
	}
}

//當連結到達上限時判斷是否等待
func Wait(b bool) PersistOption {
	return func(p *PersistOptions) {
		p.wait = b
	}
}

//Redis Dial的參數 請查看 redis包的dialOptions結構體
func DialOption(o ...redis.DialOption) PersistOption {
	return func(p *PersistOptions) {
		p.dialOption = append(p.dialOption, o...)
	}
}

type PersistOptions struct {
	//jsonfile
	filePath       string
	flieNameFormat string
	fileSize       int64
	fileMode       os.FileMode

	//sql redis 共通參數
	maxIdleConns    int
	maxOpenConns    int
	maxConnLifeTime time.Duration

	//sql
	sqlName    string        //選擇 數據庫 請參考gorm官網所支援的SQL數據庫
	sqlContent string        //請參考gorme官網 連結數據庫的篇章DSN變量
	sqlModel   []interface{} //傳入結構體 請參考gorme官網 Orm模型的建構方式

	//redis
	host       string
	dialOption []redis.DialOption
	wait       bool
}

func FileOptions(Path string, options ...PersistOption) *PersistOptions {
	p := &PersistOptions{filePath: Path}
	p.defaultFileOptions()
	for _, option := range options {
		option(p)
	}
	return p
}

func (p *PersistOptions) defaultFileOptions() {
	p.fileMode = 0664
	p.fileSize = 1024 * 1024 * 10
	p.flieNameFormat = "no.%d-%s.%s"
}

func SQLOptions(name string, content string, options ...PersistOption) *PersistOptions {
	p := &PersistOptions{
		sqlName:    name,
		sqlContent: content,
	}
	for _, option := range options {
		option(p)
	}
	return p
}

func RedisOptions(Host string, options ...PersistOption) *PersistOptions {
	p := &PersistOptions{host: Host}
	for _, option := range options {
		option(p)
	}
	return p
}
