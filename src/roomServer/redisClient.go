package roomServer

import (
	"github.com/garyburd/redigo/redis"
	"log"
	"fmt"
	. "../common"
)

type RedisClient struct {
	protocol, host, port string
	redisConn            *redis.Conn
}

func NewRedisClient() *RedisClient {
	c, err := redis.Dial(RedisProtocol, fmt.Sprintf("%s:%s", RedisHost, RedisPort))
	if err != nil {
		log.Println("Connect to redis error", err)
		return nil
	}
	return &RedisClient{protocol: RedisProtocol, host: RedisHost, port: RedisPort, redisConn: &c}
}

func (redisClient *RedisClient) GetRedisConnNotNil() redis.Conn {
	if redisClient.redisConn == nil {
		log.Println("redis connection is nil error")
		return nil
	}
	return *redisClient.redisConn
}

func (redisClient *RedisClient) close() {
	redisClient.GetRedisConnNotNil().Close()
}
