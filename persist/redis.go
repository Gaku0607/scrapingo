package persist

import (
	"encoding/json"
	"sync/atomic"

	"github.com/gomodule/redigo/redis"
)

type Redis struct {
	r *redis.Pool
	c int64
}

func newRedis(o *PersistOptions) (*Redis, error) {

	return &Redis{
		r: &redis.Pool{
			MaxActive:   o.maxOpenConns,
			MaxIdle:     o.maxIdleConns,
			IdleTimeout: o.maxConnLifeTime,
			Wait:        o.wait,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", o.host, o.dialOption...)
			},
		},
	}, nil
}

func (this *Redis) Save(item interface{}) error {
	Conn := this.r.Get()
	defer Conn.Close()
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	atomic.AddInt64(&this.c, 1)
	_, err = Conn.Do("SET", this.c, data)
	return err
}
func (this *Redis) Close() {
	this.r.Close()
}
