package persist

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

//支持MySQL PostgreSQL Sqlite3 sql Server
type Sql struct {
	DB *gorm.DB
}

func newSQL(o *PersistOptions) (*Sql, error) {
	db, err := gorm.Open(o.sqlName, o.sqlContent)
	if err != nil {
		return nil, err
	}

	db.SingularTable(true)
	db.LogMode(false)
	for _, m := range o.sqlModel {
		if err = db.AutoMigrate(m).Error; err != nil {
			return nil, err
		}
	}

	db.DB().SetConnMaxLifetime(o.maxConnLifeTime)
	db.DB().SetMaxOpenConns(o.maxOpenConns)
	db.DB().SetMaxIdleConns(o.maxIdleConns)
	return &Sql{
		DB: db,
	}, nil
}

func (this *Sql) Save(item interface{}) error {
	return this.DB.Create(item).Error
}
func (this *Sql) Close() {
	this.DB.Close()
}
