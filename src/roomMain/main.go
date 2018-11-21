package main

import (
	"flag"
	"log"
	//. "../roomServer"
	//"github.com/garyburd/redigo/redis"
	"fmt"
)

var tls = flag.Bool("tls", true, "whether TLS is used")
var port = flag.Int("port", 9002, "The TCP port that the server listens on")
var colliderSrv = flag.String("collider", "http://192.168.1.30:8080/", "The origin of the collider")

func main() {
	flag.Parse()

	log.Printf("Starting roomServer: tls = %t, port = %d, collider=%s", *tls, *port, *colliderSrv)
	var a map[string]string
	a = make(map[string]string)
	fmt.Println(a["1"])
	//redisClient := NewRedisClient()
	//redisCon := redisClient.GetRedisConnNotNil()
	//fmt.Println(redisCon.Do("WATCH", "roomid111"))
	//if roomValue, error := redis.StringMap(redisCon.Do("HGETALL", "roomid111")); error == nil {
	//	fmt.Println(roomValue)
	//	occupancy := len(roomValue)
	//	fmt.Printf("occupancy:%d \n", occupancy)
	//}else {
	//	fmt.Println(error)
	//}
	//fmt.Println(redisCon.Do("MULTI"))
	//fmt.Println(redisCon.Do("HSETNX", "roomid111", "clientid111", "clientid111Json"))
	//fmt.Println(redisCon.Do("EXEC"))
}
