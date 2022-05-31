package dao

import (
	"dube/internal/cat/options"
	"fmt"
	log "github.com/golang/glog"
	"github.com/gomodule/redigo/redis"
	"time"
)

type Dao struct {
	redis       *redis.Pool
	redisExpire int
}

func New(c *options.Redis) *Dao {
	d := &Dao{
		redis: NewRedis(c),
	}
	d.redisExpire = int(time.Duration(c.Expire) / time.Second)
	return d
}

func (d *Dao) Close() {
	d.redis.Close()
}

func NewRedis(c *options.Redis) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     c.Idle,
		IdleTimeout: time.Duration(c.IdleTimeout),
		MaxActive:   c.Active,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial(
				c.Network,
				c.Addr,
				redis.DialConnectTimeout(time.Duration(c.DialTimeout)),
				redis.DialReadTimeout(time.Duration(c.ReadTimeout)),
				redis.DialWriteTimeout(time.Duration(c.WriteTimeout)),
				//redis.DialPassword(c.Auth),
			)
			if err != nil {
				return nil, err
			}
			return conn, nil
		},
	}
}

func KeyMidServer(mid int64) string {
	return fmt.Sprintf("mid:%d", mid)
}

func KeyKeyServer(key string) string {
	return fmt.Sprintf("key:%s", key)
}

func (d *Dao) serversByKeys(keys []string) {
	r := d.redis.Get()
	defer r.Close()

}

func (d *Dao) ExpireMapping(mid int64, key string) (bool, error) {
	r := d.redis.Get()
	defer r.Close()

	n := 1

	if mid > 0 {
		if err := r.Send("EXPIRE", KeyMidServer(mid), d.redisExpire); err != nil {
			return false, err
		}
		n++
	}

	if err := r.Send("EXPIRE", KeyKeyServer(key), d.redisExpire); err != nil {
		return false, err
	}

	if err := r.Flush(); err != nil {
		return false, err
	}

	for i := 0; i < n; i++ {
		if b, err := redis.Bool(r.Receive()); err != nil {
			return b, err
		}
	}

	return true, nil
}

func (d *Dao) AddMapping(mid int64, key, server string) error {
	r := d.redis.Get()
	defer r.Close()

	n := 2

	if mid > 0 {
		if err := r.Send("HSET", KeyMidServer(mid), key, server); err != nil {
			log.Errorf("redis send HSET(%s,%s,%s) error - (%v)", KeyMidServer(mid), key, server, err)
			return err
		}

		if err := r.Send("EXPIRE", KeyMidServer(mid), d.redisExpire); err != nil {
			log.Errorf("redis send EXPIRE(%s,%d) error - (%v)", KeyMidServer(mid), d.redisExpire)
			return err
		}
		n += 2
	}

	if err := r.Send("SET", KeyKeyServer(key), server); err != nil {
		log.Errorf("redis send SET(%s,%s)", KeyKeyServer(key), server)
		return err
	}

	if err := r.Send("EXPIRE", KeyKeyServer(key), d.redisExpire); err != nil {
		log.Errorf("redis send EXPIRE(%s,%d)", KeyKeyServer(key), d.redisExpire)
		return err
	}

	if err := r.Flush(); err != nil {
		return err
	}

	for i := 0; i < n; i++ {
		if _, err := r.Receive(); err != nil {
			log.Errorf("redis Receive error - (%v)", err)
			return err
		}
	}

	return nil
}
