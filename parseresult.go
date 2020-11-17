package scrapingo

//解析格式
type ParseFunc func([]byte) *ParseResult

func NilParse(b []byte) *ParseResult {
	return &ParseResult{}
}

//解析後的結果
//Items為解析後 自定義的返回值
//Requests為解析後所返回的下次請求
type ParseResult struct {
	ParentRequest *Request
	Requests      []*Request
	Items         []interface{}
}
