package roomServer

import (
	"time"
	"fmt"
	"math/rand"
	"encoding/json"
	. "common"
	"testing"
)

func TestMakePcConstraints(t *testing.T){
	result := makePcConstraints("False","True","True")
	fmt.Println(result)
	jsonByte,_ :=json.Marshal(result)
	fmt.Println(string(jsonByte))
}

func TestRoomServer_AddClient2Room(t *testing.T) {
	roomServer := NewRoomServer()
	roomid := "roomid"
	requestCount := 10
	var complate = make(chan int, requestCount)
	for i := 0; i < requestCount; i++ {
		go func(i int) {
			sleepS := rand.Int63n(10)
			time.Sleep(time.Duration(sleepS) * time.Second)
			result := roomServer.AddClient2Room(roomid, fmt.Sprintf("clientid%d", i))
			if value, error := json.Marshal(result); error == nil {
				Info.Printf(string(value))
			}
			complate <- 1
		}(i)
	}

	for i := 0; i < requestCount; i++ {
		<-complate
	}
}
