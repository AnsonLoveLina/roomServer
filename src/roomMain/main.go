package main

import (
	"flag"
	. "common"
	. "roomServer"
)

var tls = flag.Bool("tls", true, "whether TLS is used")
var port = flag.Int("port", 8080, "The TCP port that the server listens on")

func main() {
	flag.Parse()
	Info.Printf("Starting roomServer: tls = %t, port = %d", *tls, *port)
	roomServer := NewRoomServer()
	roomServer.Run(*port,*tls)
}
