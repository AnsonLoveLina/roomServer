package server

import (
	"github.com/garyburd/redigo/redis"
	. "common"
	"time"
)

var RedisClient *redis.Pool

func init() {
	RedisClient = &redis.Pool{
		MaxIdle:100,
		MaxActive:1024,
		IdleTimeout:180*time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", RedisHost+":"+RedisPort)
			if err != nil {
				return nil, err
			}
			// 选择db
			//c.Do("SELECT", REDIS_DB)
			return c, nil
		},
	}
}