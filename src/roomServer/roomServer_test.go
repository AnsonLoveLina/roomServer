package roomServer

import (
	"time"
	"fmt"
	"math/rand"
	"encoding/json"
	. "common"
	"testing"
)

func Test(t *testing.T){
	a := map[string]interface{}{"gitHash": "", "time": "", "branch": ""}
	fmt.Println(JsonByte(json.Marshal(a)))

}

func TestAddClient2Room(t *testing.T) {
	roomid := "roomid"
	requestCount := 10
	var complate = make(chan int, requestCount)
	for i := 0; i < requestCount; i++ {
		go func(i int) {
			sleepS := rand.Int63n(10)
			time.Sleep(time.Duration(sleepS) * time.Second)
			result := AddClient2Room(roomid, fmt.Sprintf("clientid%d", i))
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
