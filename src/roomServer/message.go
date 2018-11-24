package roomServer

import (
	"net/http"
	. "common"
	"github.com/gorilla/mux"
	"github.com/garyburd/redigo/redis"
	"encoding/json"
)

func (rs *RoomServer) messageRoomHandler(rw http.ResponseWriter, r *http.Request) {
	_, requestBody := GetRequestJson(r)
	roomid := mux.Vars(r)["roomid"]
	clientid := mux.Vars(r)["clientid"]
	saveMessageFromClient(roomid, clientid, requestBody)
}

type messageResult struct {
	Error string `json:"error"`
	Saved bool   `json:"saved"`
}

func saveMessageFromClient(roomid, clientid string, requestBody string) (messageResult) {
	//先用clientid作为redis的clientKey
	var clientKey = clientid
	var redisCon = RedisClient.Get()
	for i := 0; ; i++ {
		var roomValue map[string]string
		var error error
		if result, err := redis.String(redisCon.Do("WATCH", roomid)); err != nil || result != "OK" {
			Error.Printf("command:WATCH %s , result:%s , error:%s", roomid, result, err)
			goto continueFlag
		}
		if roomValue, error = redis.StringMap(redisCon.Do("HGETALL", roomid)); error != nil {
			Error.Printf("command:HGETALL %s , error:%s", roomid, error)
			goto continueFlag
		} else if roomValue == nil {
			Warn.Printf("Unknow room:%s", roomid)
			return messageResult{RESPONSE_UNKNOWN_ROOM, false}
		} else if roomValue[clientKey] == "" {
			Warn.Printf("Unknow client:%s", clientKey)
			return messageResult{RESPONSE_UNKNOWN_CLIENT, false}
		} else if len(roomValue) >= roomMaxOccupancy {
			return messageResult{"", false}
		} else {
			clientJson := roomValue[clientKey]
			var otherClient Client
			json.Unmarshal([]byte(clientJson), otherClient)
			otherClient.Message = append(otherClient.Message, requestBody)

			if newClient, error := json.Marshal(&otherClient); error == nil {
				roomValue[clientKey] = string(newClient[:])
			} else {
				Error.Println(error)
				return messageResult{"", false}
			}
		}

		if result, error := redis.String(redisCon.Do("MULTI")); error != nil || result != "OK" {
			Error.Printf("command:MULTI , result:%s , error:%s", result, error)
			goto continueFlag
		}
		if result, error := redis.String(redisCon.Do("HSETNX", roomid, clientKey, roomValue[clientKey])); error != nil || result != "QUEUED" {
			Error.Printf("command:HSETNX %s %s %s , result:%s , error:%s", roomid, clientKey, roomValue[clientKey], result, error)
			goto continueFlag
		}
		if result, error := redisCon.Do("EXEC"); error != nil {
			Error.Printf("command:EXEC , result:%d , error:%s", result, error)
			goto continueFlag
		} else if result != nil {
			Info.Printf("success client: %s to Room: %s add,retries:%d!", clientKey, roomid,i)
			return messageResult{"", true}
		} else {
			goto continueFlag
		}

		continueFlag:
			Info.Printf("db cas cause bad client: %s to Room: %s message", clientKey, roomid)
		if i < errorBreakMax {
			break
		} else {
			continue
		}
	}
}
