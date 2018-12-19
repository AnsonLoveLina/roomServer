package main

import (
	"flag"
	. "server"
	log "github.com/sirupsen/logrus"
	"github.com/mattn/go-colorable"
)

var tls = flag.Bool("tls", true, "whether TLS is used")
var port = flag.Int("port", 8080, "The TCP port that the server listens on")

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
	log.Infof("Starting server: tls = %t, port = %d", *tls, *port)
	roomServer := NewRoomServer()
	roomServer.Run(*port, *tls)
}
