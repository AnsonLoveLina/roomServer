package main

import (
	"flag"
	. "common"
)

var tls = flag.Bool("tls", true, "whether TLS is used")
var port = flag.Int("port", 9002, "The TCP port that the server listens on")
var colliderSrv = flag.String("collider", "http://192.168.1.30:8080/", "The origin of the collider")

func main() {
	flag.Parse()

	Info.Printf("Starting roomServer: tls = %t, port = %d, collider=%s", *tls, *port, *colliderSrv)
}
