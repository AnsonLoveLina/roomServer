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
	a := make([]string,0)
	fmt.Printf("len:%d,cap:%d \n",len(a),cap(a))
	a = append(a,"s")
	a = append(a,"s")
	a = append(a,"s")
	a = append(a,"s")
	fmt.Println(a)
	fmt.Printf("len:%d,cap:%d",len(a),cap(a))

	b := [2]string{}
	b[0] = "0"
	b[1] = "1"
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
