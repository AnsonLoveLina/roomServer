package server

import (
	"net/http"
	"github.com/gorilla/mux"
	. "roomServer/common"
	"github.com/garyburd/redigo/redis"
	"github.com/sirupsen/logrus"
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
		logrus.Infof("room:%s has state %s", roomid, result.roomState)
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
			logrus.WithFields(logrus.Fields{"result": result, "error": error}).Errorf("command:WATCH %s", roomid)
			goto continueFlag
		}
		if roomValue, error = ClientMap(redisCon.Do("HGETALL", roomid)); error != nil {
			logrus.WithFields(logrus.Fields{"error": error}).Errorf("command:HGETALL %s", roomid)
			goto continueFlag
		} else if roomValue == nil {
			logrus.WithFields(logrus.Fields{"roomid": roomid}).Warn("Unknow room")
			return leaveResult{RESPONSE_UNKNOWN_ROOM, ""}
		} else if roomValue[clientKey] == nil {
			logrus.WithFields(logrus.Fields{"clientKey": clientKey}).Warn("Unknow client")
			return leaveResult{RESPONSE_UNKNOWN_CLIENT, ""}
		}

		delete(roomValue, clientKey)
		if len(roomValue) > 0 {
			otherClient := getOtherClient(roomValue, clientKey)
			roomState = string(MarshalNoError(*otherClient, []byte{}))
		}

		if result, error := redis.String(redisCon.Do("MULTI")); error != nil || result != "OK" {
			logrus.WithFields(logrus.Fields{"result": result, "error": error}).Error("command:MULTI")
			goto continueFlag
		}
		if result, error := redis.String(redisCon.Do("HDEL", roomid, clientKey)); error != nil || result != "QUEUED" {
			logrus.WithFields(logrus.Fields{"result": result, "error": error}).Errorf("command:HSETNX %s %s", roomid, clientKey)
			goto continueFlag
		}
		if result, error := redisCon.Do("EXEC"); error != nil {
			logrus.WithFields(logrus.Fields{"result": result, "error": error}).Error("command:EXEC")
			goto continueFlag
		} else if result != nil {
			logrus.WithFields(logrus.Fields{"result": result, "client": clientKey, "Room": roomid, "retries": i}).Info("client success leave to the room")
			return leaveResult{"", roomState}
		} else {
			goto continueFlag
		}
	continueFlag:
		logrus.WithFields(logrus.Fields{"client": clientKey, "Room": roomid}).Info("db cas cause client bad leave to the room")
		if i < errorBreakMax {
			break
		} else {
			continue
		}
	}
	return
}
