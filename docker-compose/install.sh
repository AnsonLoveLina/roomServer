#!/bin/bash
cd /go/src && 
#gopm get -g golang.org/x/crypto && 
#gopm get -g golang.org/x/sys/unix && 
#gopm get -g github.com/stretchr/testify && 
go get -v ./... && 
roomServer  -tls=false -iceServerUrl=http://192.168.1.95:8080 -wsHost=192.168.1.95:8089 -redisHost=redis
