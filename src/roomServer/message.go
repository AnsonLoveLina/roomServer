package roomServer

import (
	"net/http"
	. "common"
	"github.com/gorilla/mux"
	"github.com/garyburd/redigo/redis"
	"encoding/json"
	"fmt"
	"strings"
)

type messageResult struct {
	Error string `json:"error"`
	Saved bool   `json:"saved"`
}

func (rs *RoomServer) messageRoomHandler(rw http.ResponseWriter, r *http.Request) {
	requestJson, requestBody := GetRequestJson(r)
	roomid := mux.Vars(r)["roomid"]
	clientid := mux.Vars(r)["clientid"]
	result := SaveMessageFromClient(roomid, clientid, requestBody)
	if result.Error != "" {
		messageWriteResponse(rw, result.Error)
	}

	if ! result.Saved {
		sendMessageToCollider(rw, roomid, clientid, requestJson, requestBody)
	} else {
		messageWriteResponse(rw, RESPONSE_SUCCESS)
	}
}

func sendMessageToCollider(rw http.ResponseWriter, roomid, clientid string, requestJson map[string]interface{}, requestBody string) {
	Info.Printf("Forwarding message to collider for room %s client %s", roomid, clientid)
	_, wssPostUrl := getWssParameters(requestJson)
	url := wssPostUrl + "/" + roomid + "/" + clientid
	req, err := http.NewRequest("POST", url, strings.NewReader(requestBody))
	if err != nil {
		Error.Printf("Failed to send message to collider: %s", err)
		return
	}
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		Error.Printf("Failed to send message to collider: %s", err)
		return
	}

	if response.StatusCode != 200 {
		Error.Printf("Failed to send message to collider: %d", response.StatusCode)
		return
	}

	messageWriteResponse(rw, RESPONSE_SUCCESS)
}

func SaveMessageFromClient(roomid, clientid string, requestBody string) (result messageResult) {
	//先用clientid作为redis的clientKey
	var clientKey = clientid
	var roomValue map[string]*Client
	var redisCon = RedisClient.Get()
	defer redisCon.Close()
	for i := 0; ; i++ {
		var error error
		if result, err := redis.String(redisCon.Do("WATCH", roomid)); err != nil || result != "OK" {
			Error.Printf("command:WATCH %s , result:%s , error:%s", roomid, result, err)
			goto continueFlag
		}
		if roomValue, error = ClientMap(redisCon.Do("HGETALL", roomid)); error != nil {
			Error.Printf("command:HGETALL %s , error:%s", roomid, error)
			goto continueFlag
		} else if roomValue == nil {
			Warn.Printf("Unknow room:%s", roomid)
			return messageResult{RESPONSE_UNKNOWN_ROOM, false}
		} else if roomValue[clientKey] == nil {
			Warn.Printf("Unknow client:%s", clientKey)
			return messageResult{RESPONSE_UNKNOWN_CLIENT, false}
		} else if len(roomValue) >= roomMaxOccupancy {
			return messageResult{"", false}
		} else {
			client := roomValue[clientKey]
			client.Message = append(client.Message, requestBody)

			roomValue[clientKey] = client
		}

		if result, error := redis.String(redisCon.Do("MULTI")); error != nil || result != "OK" {
			Error.Printf("command:MULTI , result:%s , error:%s", result, error)
			goto continueFlag
		}
		if result, error := redis.String(redisCon.Do("HSETNX", roomid, clientKey, MarshalNoErrorStr(*roomValue[clientKey], ""))); error != nil || result != "QUEUED" {
			Error.Printf("command:HSETNX %s %s %s , result:%s , error:%s", roomid, clientKey, MarshalNoErrorStr(*roomValue[clientKey], ""), result, error)
			goto continueFlag
		}
		if result, error := redisCon.Do("EXEC"); error != nil {
			Error.Printf("command:EXEC , result:%d , error:%s", result, error)
			goto continueFlag
		} else if result != nil {
			Info.Printf("success client: %s to Room: %s add,retries:%d!", clientKey, roomid, i)
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
	return
}

func messageWriteResponse(rw http.ResponseWriter, result string) {
	responseObj := map[string]interface{}{"result": result}
	response, error := json.Marshal(responseObj)
	if error != nil {
		err := fmt.Sprintf("json marshal error %s result:%s", error.Error(), result)
		Error.Panicln(err)
		response, _ = json.Marshal(map[string]interface{}{"result": err, "params": make(map[string]interface{})})
	}
	Debug.Printf("message response:%s", string(response))
	rw.Write(response)
}
