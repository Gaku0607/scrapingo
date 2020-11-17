package scrapingo

//在自定義的結構體中中添加Model 例：
//  type Scrapingo struct{
//      scrapingo.Model
//      ....
//  }
// or 取Model其中的參數
//  type Scrapingo struct{
//      ItemID int
//      ....
//      ParentID int
//      ....
//  }
//在爬取時則會自動追加所對應的值

type Model struct {
	//唯一識別
	ItemID int `gorm:"primary_key" json:"itemID"`
	//ParentRequestID
	ParentID int `json:"parentID"`
	//ParentRequestURL
	ParentURL string `json:"parentURL"`
}
