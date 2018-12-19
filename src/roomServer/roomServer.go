package main

import (
	"flag"
	"github.com/mattn/go-colorable"
	log "github.com/sirupsen/logrus"
	. "roomServer/common"
	. "roomServer/server"
)

var tls = flag.Bool("tls", true, "whether TLS is used")
var port = flag.Int("port", 8080, "The TCP port that the server listens on")
var redisHost = flag.String("redisHost", "127.0.0.1", "The redisHost that the server used")
var iceServerUrl = flag.String("iceServerUrl", "http://192.168.1.21:8080", "The iceServerUrl that the server used")
var wsHost = flag.String("wsHost", "192.168.1.21:8089", "The wsHost that the server used")

func init() {
	// Log as JSON instead of the default ASCII formatter.
	//log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(colorable.NewColorableStdout())

	//log.SetReportCaller(true)

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)
}

func main() {
	flag.Parse()
	log.WithFields(log.Fields{"redisHost": *redisHost, "iceServerUrl": *iceServerUrl, "wsHost": *wsHost}).Infof("Starting server: tls = %t, port = %d", *tls, *port)
	RedisHost = *redisHost
	ICE_SERVER_BASE_URL = *iceServerUrl
	WSS_INSTANCES[0][WSS_INSTANCE_HOST_KEY] = *wsHost
	roomServer := NewRoomServer()
	roomServer.Run(*port, *tls)
}
