package persist

const (
	SQL      = "sql"
	JSONFILE = "jsonfile"
	REDIS    = "redis"
)

type Persist interface {
	Save(interface{}) error
	Close()
}
type NilPersist struct {
}

func (n *NilPersist) Save(interface{}) error { return nil }
func (n *NilPersist) Close()                 { return }

func NewPersistStore(format string, Options *PersistOptions) (Persist, error) {
	switch format {
	case SQL:
		return newSQL(Options)
	case JSONFILE:
		return newJSONFile(Options)
	case REDIS:
		return newRedis(Options)
	default:
		return &NilPersist{}, nil
	}
}
