package roomServer

import (
	"net/http"
	. "common"
	"github.com/gorilla/mux"
	"github.com/garyburd/redigo/redis"
	"encoding/json"
	"fmt"
	"strings"
	"github.com/sirupsen/logrus"
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
	logrus.WithFields(logrus.Fields{"room": roomid, "clientid": clientid}).Info("Forwarding message to collider")
	_, wssPostUrl := getWssParameters(requestJson)
	url := wssPostUrl + "/" + roomid + "/" + clientid
	req, err := http.NewRequest("POST", url, strings.NewReader(requestBody))
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("Failed to send message to collider")
		return
	}
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("Failed to send message to collider")
		return
	}

	if response.StatusCode != 200 {
		logrus.WithFields(logrus.Fields{"StatusCode": response.StatusCode}).Error("Failed to send message to collider")
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
			logrus.WithFields(logrus.Fields{"result": result, "error": err}).Errorf("command:WATCH %s", roomid)
			goto continueFlag
		}
		if roomValue, error = ClientMap(redisCon.Do("HGETALL", roomid)); error != nil {
			logrus.WithFields(logrus.Fields{"result": roomValue, "error": error}).Errorf("HGETALL:WATCH %s", roomid)
			goto continueFlag
		} else if roomValue == nil {
			logrus.WithFields(logrus.Fields{"roomid": roomid}).Warn("Unknow room")
			return messageResult{RESPONSE_UNKNOWN_ROOM, false}
		} else if roomValue[clientKey] == nil {
			logrus.WithFields(logrus.Fields{"clientKey": clientKey}).Warn("Unknow client")
			return messageResult{RESPONSE_UNKNOWN_CLIENT, false}
		} else if len(roomValue) >= roomMaxOccupancy {
			return messageResult{"", false}
		} else {
			client := roomValue[clientKey]
			client.Message = append(client.Message, requestBody)

			roomValue[clientKey] = client
		}

		if result, error := redis.String(redisCon.Do("MULTI")); error != nil || result != "OK" {
			logrus.WithFields(logrus.Fields{"result": result, "error": error}).Error("command:MULTI")
			goto continueFlag
		}
		if result, error := redis.String(redisCon.Do("HSET", roomid, clientKey, MarshalNoErrorStr(*roomValue[clientKey], ""))); error != nil || result != "QUEUED" {
			logrus.WithFields(logrus.Fields{"result": result, "error": error}).Errorf("command:HSET %s %s %s", roomid, clientKey, MarshalNoErrorStr(*roomValue[clientKey], ""))
			goto continueFlag
		}
		if result, error := redis.Ints(redisCon.Do("EXEC")); error != nil {
			logrus.WithFields(logrus.Fields{"result": result, "error": error}).Error("command:EXEC")
			goto continueFlag
		} else if result != nil && result[0] == 0 {
			logrus.WithFields(logrus.Fields{"result":result,"client": clientKey, "Room": roomid, "retries": i}).Info("client success message to the room")
			return messageResult{"", true}
		} else {
			goto continueFlag
		}

	continueFlag:
		logrus.WithFields(logrus.Fields{"client": clientKey, "Room": roomid}).Info("db cas cause client bad message to the room")
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
		logrus.WithFields(logrus.Fields{"result": result, "error": error.Error()}).Error("json marshal error")
		response, _ = json.Marshal(map[string]interface{}{"result": err, "params": make(map[string]interface{})})
	}
	logrus.WithFields(logrus.Fields{"response": string(response)}).Debug("message success response")
	rw.Write(response)
}
