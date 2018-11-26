package roomServer

import (
	"net/http"
	"strconv"
	"github.com/gorilla/mux"
	"fmt"
	"strings"
	"encoding/json"
	"math/rand"
	. "common"
	"github.com/garyburd/redigo/redis"
)

func (rs *RoomServer) joinRoomHandler(rw http.ResponseWriter, r *http.Request) {
	roomid := mux.Vars(r)["roomid"]

	var clientid = strconv.FormatInt(rand.Int63n(89999999)+10000000, 10)
	requestJson, _ := GetRequestJson(r)
	if requestJson["clientid"] != nil {
		clientid = Interface2string(requestJson["clientid"], "")
	}

	result := AddClient2Room(roomid, clientid)
	if result.Error != "" {
		Error.Printf("Error adding client to Room: %s, room_state=%s",
			result.Error, result.Room)
		joinWriteResponse(rw, result.Error, make(map[string]interface{}), make([]string, 0))
		return
	}

	params := getRoomParameters(r, requestJson, roomid, clientid, result.IsInitiator)
	joinWriteResponse(rw, "SUCCESS", params, result.Messages)
}

type joinResult struct {
	Error       string   `json:"error"`
	IsInitiator bool     `json:"IsInitiator"`
	Messages    []string `json:"Messages"`
	Room        Room     `json:"room_state"`
}

func AddClient2Room(roomid string, clientid string) (result joinResult) {
	//先用clientid作为redis的clientKey
	var clientKey = clientid
	var isInitiator = false
	//切片保证容错性
	var messages = make([]string, 0, roomMaxOccupancy)
	//roomKey := fmt.Sprintf("%s/%s", request.URL.Host, roomid)
	var occupancy int
	var roomValue map[string]*Client
	var room Room
	var redisCon = RedisClient.Get()
	defer redisCon.Close()
	for i := 0; ; i++ {
		var error error
		var client string
		if result, err := redis.String(redisCon.Do("WATCH", roomid)); err != nil || result != "OK" {
			Error.Printf("command:WATCH %s , result:%s , error:%s", roomid, result, err)
			goto continueFlag
		}
		if roomValue, error = ClientMap(redisCon.Do("HGETALL", roomid)); error != nil {
			Error.Printf("command:HGETALL %s , error:%s", roomid, error)
			goto continueFlag
		}
		room = Room{Clients: roomValue}
		//json.Unmarshal(roomValue,clients)
		occupancy = len(roomValue)

		if occupancy >= roomMaxOccupancy {
			return joinResult{Error: roomFull.getString()}
		}
		if value := roomValue[clientKey]; value != nil {
			return joinResult{Error: duplicateClient.getString()}
		}
		if occupancy == 0 { //the first client of this Room
			isInitiator = true
			room = Room{Clients: map[string]*Client{clientKey: NewClient(isInitiator)}}
		} else {
			isInitiator = false
			var i = 0
			for _, client := range roomValue {
				messages = append(messages, client.Message...)
				i++
				//是否应该clean client message
				client.Message = make([]string, 0, 10)
			}
			if newClient, error := json.Marshal(NewClient(isInitiator)); error == nil {
				client = string(newClient[:])
				room.Clients[clientKey] = NewClient(isInitiator)
			} else {
				Error.Println(error)
				goto continueFlag
			}
		}

		if result, error := redis.String(redisCon.Do("MULTI")); error != nil || result != "OK" {
			Error.Printf("command:MULTI , result:%s , error:%s", result, error)
			goto continueFlag
		}
		if result, error := redis.String(redisCon.Do("HSETNX", roomid, clientKey, client)); error != nil || result != "QUEUED" {
			Error.Printf("command:HSETNX %s %s %s , result:%s , error:%s", roomid, clientKey, client, result, error)
			goto continueFlag
		}
		if result, error := redisCon.Do("EXEC"); error != nil {
			Error.Printf("command:EXEC , result:%d , error:%s", result, error)
			goto continueFlag
		} else if result != nil {
			Info.Printf("success client: %s to Room: %s add , retries:%d!", clientKey, roomid, i)
			return joinResult{"", isInitiator, messages, room}
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

func joinWriteResponse(rw http.ResponseWriter, result string, params map[string]interface{}, messages []string) {
	params["Messages"] = messages
	responseObj := map[string]interface{}{"result": result, "params": params}
	response, error := json.Marshal(responseObj)
	if error != nil {
		err := fmt.Sprintf("json marshal error %s result:%s,params:%s,Messages:%s", error.Error(), result, params, messages)
		Error.Panicln(err)
		response, _ = json.Marshal(map[string]interface{}{"result": err, "params": make(map[string]interface{})})
	}
	rw.Write(response)
}

func getRoomParameters(request *http.Request, requestJson map[string]interface{}, roomid string, clientid string, isInitiator interface{}) (map[string]interface{}) {
	var warningMessages = make([]string, 0, 10)
	var message string
	userAgent := request.UserAgent()
	iceTransports := Interface2string(requestJson["it"], "")
	iceServerTransports := Interface2string(requestJson["tt"], "")
	var iceServerBaseUrl string
	if requestJson["ts"] != nil {
		iceServerBaseUrl = Interface2string(requestJson["ts"], "")
	} else {
		iceServerBaseUrl = ICE_SERVER_BASE_URL
	}

	audio := Interface2string(requestJson["audio"], "")
	video := Interface2string(requestJson["video"], "")

	firefoxFakeDevice := Interface2string(requestJson["firefox_fake_device"], "")

	hd := strings.ToLower(Interface2string(requestJson["hd"], ""))

	if hd != "" && video != "" {
		message = "The \"hd\" parameter has overridden video=" + video
		Info.Println(message)
		warningMessages = append(warningMessages, message)
	}
	if hd == "true" {
		video = "mandatory:minWidth=1280,mandatory:minHeight=720"
	} else if hd != "" && video != "" && getHdDefault(userAgent) {
		video = "optional:minWidth=1280,optional:minHeight=720"
	}

	if requestJson["minre"] != nil || requestJson["maxre"] != nil {
		message = "The \"minre\" and \"maxre\" parameters are no longer supported. Use \"video\" instead."
		Info.Println(message)
		warningMessages = append(warningMessages, message)
	}

	dtls := Interface2string(requestJson["dtls"], "")
	dscp := Interface2string(requestJson["dscp"], "")
	ipv6 := Interface2string(requestJson["ipv6"], "")

	var iceServerUrl = ""
	if len(iceServerBaseUrl) > 0 {
		iceServerUrl = fmt.Sprintf(ICE_SERVER_URL_TEMP, ICE_SERVER_BASE_URL, ICE_SERVER_API_KEY)
	}

	iceServerOverride := ICE_SERVER_OVERRIDE
	pcConfig := makePcConfig(iceTransports, iceServerOverride)
	pcConstraints := makePcConstraints(dtls, dscp, ipv6)
	offer_options := struct{}{}
	mediaConstraints := makeMediaStreamConstraints(audio, video, firefoxFakeDevice)
	wssUrl, wssPostUrl := getWssParameters(requestJson)

	bypassJoinConfirmation := false
	params := map[string]interface{}{
		"error_messages":           []string{},
		"warning_messages":         warningMessages,
		"pc_config":                JsonByte(json.Marshal(pcConfig)),
		"pc_constraints":           JsonByte(json.Marshal(pcConstraints)),
		"offer_options":            JsonByte(json.Marshal(offer_options)),
		"media_constraints":        JsonByte(json.Marshal(mediaConstraints)),
		"ice_server_url":           iceServerUrl,
		"ice_server_transports":    iceServerTransports,
		"wss_url":                  wssUrl,
		"wss_post_url":             wssPostUrl,
		"bypass_join_confirmation": JsonByte(json.Marshal(bypassJoinConfirmation)),
		"version_info":             JsonByte(json.Marshal(getVersionInfo())),
		//"include_rtstats_js" :      include_rtstats_js,
	}

	if roomid != "" {
		params["room_id"] = roomid
		var proto string
		if strings.Index(strings.ToLower(request.Proto), "https") == -1 {
			proto = "http://"
		} else {
			proto = "https://"
		}
		roomLink := proto + request.Host + "/r/" + roomid
		if request.URL.Query().Encode() != "" {
			roomLink = roomLink + "?" + request.URL.Query().Encode()
		}
		params["room_link"] = roomLink
	}
	if clientid != "" {
		params["client_id"] = clientid
	}
	if isInitiator != nil {
		params["is_initiator"] = isInitiator
	}

	return params
}
func getVersionInfo() interface{} {
	return map[string]interface{}{"gitHash": "", "time": "", "branch": ""}
}

func makeMediaTrackConstraints(constraints string) (trackConstraints interface{}) {
	if constraints == "" || strings.ToLower(constraints) == "true" {
		trackConstraints = true
	} else if strings.ToLower(constraints) == "false" {
		trackConstraints = false
	} else {
		trackConstraints := map[string]interface{}{"optional": make([]map[string]interface{}, 0),}
		for _, constraint := range strings.Split(constraints, ",") {
			var mandatory = true
			tokens := strings.Split(constraint, ":")
			if len(tokens) == 2 {
				mandatory = tokens[0] == "mandatory"
			} else {
				mandatory = strings.HasPrefix(tokens[0], "goog")
			}

			tokens = strings.Split(tokens[len(tokens)-1], "=")
			if mandatory {
				trackConstraints["mandatory"] = map[string]interface{}{tokens[0]: tokens[1]}
			} else {
				optional := trackConstraints["optional"].([]map[string]interface{})
				optional = append(optional, map[string]interface{}{tokens[0]: tokens[1]})
			}
		}
	}
	return
}

func makeMediaStreamConstraints(audio string, video string, firefoxFakeDevice string) map[string]interface{} {
	var stream_constraints = map[string]interface{}{
		"audio": makeMediaTrackConstraints(audio),
		"video": makeMediaTrackConstraints(video),
	}

	if firefoxFakeDevice != "" {
		stream_constraints["fake"] = true
	}
	Info.Printf("Applying media constraints: %s", JsonByte(json.Marshal(stream_constraints)))
	return stream_constraints
}

func makePcConstraints(dtls string, dscp string, ipv6 string) interface{} {
	var optionals = make([]map[string]bool, 0)
	dtlsValue, dtlsError := strconv.ParseBool(dtls)
	if dtlsError == nil {
		optionals = append(optionals, map[string]bool{"DtlsSrtpKeyAgreement": dtlsValue})
	}
	dscpValue, dscpError := strconv.ParseBool(dscp)
	if dscpError == nil {
		optionals = append(optionals, map[string]bool{"googDscp": dscpValue})
	}
	ipv6Value, ipv6Error := strconv.ParseBool(ipv6)
	if ipv6Error == nil {
		optionals = append(optionals, map[string]bool{"googIPv6": ipv6Value})
	}

	return struct {
		Optional []map[string]bool `json:"optional"`
	}{Optional: optionals}

}
func makePcConfig(iceTransports string, iceServerOverride []interface{}) map[string]interface{} {
	var config = map[string]interface{}{
		"iceServers":    make([]interface{}, 0),
		"bundlePolicy":  "max-bundle",
		"rtcpMuxPolicy": "require",
	}
	if iceServerOverride != nil {
		config["iceServers"] = iceServerOverride
	}
	if iceTransports != "" {
		config["iceTransports"] = iceTransports
	}
	return config
}

func getHdDefault(userAgent string) bool {
	if strings.Contains(userAgent, "Android") || strings.Contains(userAgent, "Chrome") {
		return false
	}

	return true
}
