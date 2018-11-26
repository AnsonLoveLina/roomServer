package roomServer

import (
	"net/http"
	"github.com/gorilla/mux"
	. "common"
	"github.com/garyburd/redigo/redis"
)

type leaveResult struct {
	error     string
	roomState string
}

func (rs *RoomServer) leaveRoomHandler(rw http.ResponseWriter, r *http.Request) {
	roomid := mux.Vars(r)["roomid"]
	clientid := mux.Vars(r)["clientid"]
	result := RemoveClientFromRoom(roomid, clientid)
	if result.error == "" {
		Info.Printf("room:%s has state %s", result.error, result.roomState)
	}
}

func getOtherClient(roomValue map[string]*Client, clientid string) *Client {
	for k, v := range roomValue {
		if k != clientid {
			v.IsInitiator = true
			return v
		}
	}
	return nil
}

func RemoveClientFromRoom(roomid string, clientid string) (result leaveResult) {
	//先用clientid作为redis的clientKey
	var clientKey = clientid
	var roomValue map[string]*Client
	var redisCon = RedisClient.Get()
	defer redisCon.Close()
	for i := 0; ; i++ {
		var error error
		var roomState string
		if result, err := redis.String(redisCon.Do("WATCH", roomid)); err != nil || result != "OK" {
			Error.Printf("command:WATCH %s , result:%s , error:%s", roomid, result, err)
			goto continueFlag
		}
		if roomValue, error = ClientMap(redisCon.Do("HGETALL", roomid)); error != nil {
			Error.Printf("command:HGET %s , error:%s", roomid, error)
			goto continueFlag
		} else if roomValue == nil {
			Warn.Printf("Unknow room:%s", roomid)
			return leaveResult{RESPONSE_UNKNOWN_ROOM, ""}
		} else if roomValue[clientKey] == nil {
			Warn.Printf("Unknow client:%s", clientKey)
			return leaveResult{RESPONSE_UNKNOWN_CLIENT, ""}
		}

		delete(roomValue, clientKey)
		if len(roomValue) > 0 {
			otherClient := getOtherClient(roomValue, clientKey)
			roomState = string(MarshalNoError(*otherClient, []byte{}))
		}

		if result, error := redis.String(redisCon.Do("MULTI")); error != nil || result != "OK" {
			Error.Printf("command:MULTI , result:%s , error:%s", result, error)
			goto continueFlag
		}
		if result, error := redis.String(redisCon.Do("HDEL", roomid, clientKey)); error != nil || result != "QUEUED" {
			Error.Printf("command:HSETNX %s %s , result:%s , error:%s", roomid, clientKey, result, error)
			goto continueFlag
		}
		if result, error := redisCon.Do("EXEC"); error != nil {
			Error.Printf("command:EXEC , result:%d , error:%s", result, error)
			goto continueFlag
		} else if result != nil {
			Info.Printf("success client: %s to Room: %s remove,retries:%d!", clientKey, roomid, i)
			return leaveResult{"", roomState}
		} else {
			goto continueFlag
		}
	continueFlag:
		Info.Printf("db cas cause bad client: %s to Room: %s leave", clientKey, roomid)
		if i < errorBreakMax {
			break
		} else {
			continue
		}
	}
	return
}
