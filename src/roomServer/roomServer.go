package roomServer

import (
	"net/http"
	"github.com/gorilla/mux"
	"math/rand"
	"fmt"
	"strconv"
	"encoding/json"
	"strings"
	. "common"
	"github.com/garyburd/redigo/redis"
	"io/ioutil"
)

type ResponseType int

const roomMaxOccupancy = 2

const (
	roomFull        ResponseType = 1
	duplicateClient ResponseType = 2
	redisError      ResponseType = 3
	success         ResponseType = 0
)

func (resp ResponseType) getString() string {
	switch (resp) {
	case success:
		return "success"
	case redisError:
		return "redisError"
	case roomFull:
		return "Room is full"
	case duplicateClient:
		return "duplicate client"
	default:
		return "unknown"
	}
}

type Room struct {
	Clients map[string]string `json:"clients"`
}

type Client struct {
	IsInitiator bool   `json:"IsInitiator"`
	Message     string `json:"Message"`
}

func NewClient(isInitiator bool) *Client {
	return &Client{IsInitiator: isInitiator}
}

func NewRoom() *Room {
	return &Room{Clients: make(map[string]string)}
}

type RoomServer struct {
}

func NewRoomServer() *RoomServer {
	roomServer := &RoomServer{}
	return roomServer
}

func (rs *RoomServer) Run(p int, tls bool) {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/join/{roomid}", rs.joinRoomHandler).Methods("POST")

	http.ListenAndServe("0.0.0.0:"+strconv.Itoa(p), router)
}

func GetRequestJson(r *http.Request) map[string]string {
	body, _ := ioutil.ReadAll(r.Body)
	requestJson := make(map[string]string)
	r.Body.Close()
	if err := json.Unmarshal(body, &requestJson); err == nil {
		Info.Println(requestJson)
	} else {
		Error.Println(err)
	}
	return requestJson
}

func (rs *RoomServer) joinRoomHandler(rw http.ResponseWriter, r *http.Request) {
	roomid := mux.Vars(r)["roomid"]
	var clientid = strconv.Itoa(rand.Int())
	requestJson := GetRequestJson(r)
	if requestJson["clientid"] != "" {
		clientid = requestJson["clientid"]
	}

	result := rs.AddClient2Room(roomid, clientid)
	if result.Error != "" {
		Error.Printf("Error adding client to Room: %s, room_state=%s",
			result.Error, result.Room)
		writeResponse(rw, result.Error, make(map[string]interface{}), make([]string, 0))
		return
	}

	params := getRoomParameters(r, requestJson, roomid, clientid, result.IsInitiator)
	writeResponse(rw, "SUCCESS", params, result.Messages)
}

type result struct {
	Error       string   `json:"error"`
	IsInitiator bool     `json:"IsInitiator"`
	Messages    []string `json:"Messages"`
	Room        Room     `json:"room_state"`
}

var errorBreakMax = 10

func (rs *RoomServer) AddClient2Room(roomid string, clientid string) (result) {
	//先用clientid作为redis的clientKey
	var clientKey = clientid
	var isInitiator = false
	var messages = make([]string, roomMaxOccupancy)
	//roomKey := fmt.Sprintf("%s/%s", request.URL.Host, roomid)
	var occupancy int
	var roomValue map[string]string
	var room Room
	var redisCon = RedisClient.Get()
	defer redisCon.Close()
	var errorMessage = fmt.Sprintf("error client: %s to Room: %s add", clientid, roomid)
	for i := 0; ; i++ {
		if result, error := redis.String(redisCon.Do("WATCH", roomid)); error != nil || result != "OK" {
			Error.Printf("command:WATCH %s , result:%s , error:%s", roomid, result, error)
			if i < errorBreakMax {
				break
			}
		}
		var error error
		if roomValue, error = redis.StringMap(redisCon.Do("HGETALL", roomid)); error != nil {
			Error.Printf("command:HGETALL %s , error:%s", roomid, error)
			if i < errorBreakMax {
				break
			}
		}
		room = Room{Clients: roomValue}
		//json.Unmarshal(roomValue,clients)
		occupancy = len(roomValue)

		if occupancy >= roomMaxOccupancy {
			return result{Error: roomFull.getString()}
		}
		if value := roomValue[clientid]; value != "" {
			return result{Error: duplicateClient.getString()}
		}

		var client string
		if occupancy == 0 { //the first client of this Room
			isInitiator = true
			if newClient, error := json.Marshal(NewClient(isInitiator)); error == nil {
				client = string(newClient[:])
				room = Room{Clients: map[string]string{clientid: client}}
			} else {
				Error.Println(error)
				if i < errorBreakMax {
					break
				}
			}
		} else {
			isInitiator = false
			var i = 0
			for _, clientJson := range roomValue {
				var otherClient Client
				json.Unmarshal([]byte(clientJson), otherClient)
				messages[i] = otherClient.Message
				i++
				//是否应该clean client message
				otherClient.Message = ""
			}
			if newClient, error := json.Marshal(NewClient(isInitiator)); error == nil {
				client = string(newClient[:])
				room.Clients[clientid] = client
			} else {
				Error.Println(error)
				if i < errorBreakMax {
					break
				}
			}
		}

		if result, error := redis.String(redisCon.Do("MULTI")); error != nil || result != "OK" {
			Error.Printf("command:MULTI , result:%s , error:%s", result, error)
			if i < errorBreakMax {
				break
			}
		}
		if result, error := redis.String(redisCon.Do("HSETNX", roomid, clientKey, client)); error != nil || result != "QUEUED" {
			Error.Printf("command:HSETNX %s %s %s , result:%s , error:%s", roomid, clientKey, client, result, error)
			if i < errorBreakMax {
				break
			}
		}
		if result, error := redisCon.Do("EXEC"); error != nil {
			Error.Printf("command:EXEC , result:%d , error:%s", result, error)
			if i < errorBreakMax {
				break
			}
		} else if result != nil {
			Info.Printf("success client: %s to Room: %s add", clientid, roomid)
			errorMessage = ""
			break
		} else {
			Info.Printf("db cas cause bad client: %s to Room: %s add.system will retry!", clientid, roomid)
		}
	}
	return result{errorMessage, isInitiator, messages, room}
}

func writeResponse(rw http.ResponseWriter, result string, params map[string]interface{}, messages []string) {
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

func get_hd_default(userAgent string) bool {
	if strings.Contains(userAgent, "Android") || strings.Contains(userAgent, "Chrome") {
		return false
	}

	return true
}

func getRoomParameters(request *http.Request, requestJson map[string]string, roomid string, clientid string, isInitiator interface{}) (map[string]interface{}) {
	var warningMessages = make([]string, 10)
	var message string
	userAgent := request.UserAgent()
	iceTransports := requestJson["it"]
	iceServerTransports := requestJson["tt"]
	var iceServerBaseUrl string
	if requestJson["ts"] != "" {
		iceServerBaseUrl = requestJson["ts"]
	} else {
		iceServerBaseUrl = ICE_SERVER_BASE_URL
	}

	audio := requestJson["audio"]
	video := requestJson["video"]

	firefoxFakeDevice := requestJson["firefox_fake_device"]

	hd := strings.ToLower(requestJson["hd"])

	if hd != "" && video != "" {
		message = "The \"hd\" parameter has overridden video=" + video
		Info.Println(message)
		warningMessages = append(warningMessages, message)
	}
	if hd == "true" {
		video = "mandatory:minWidth=1280,mandatory:minHeight=720"
	} else if hd != "" && video != "" && get_hd_default(userAgent) {
		video = "optional:minWidth=1280,optional:minHeight=720"
	}

	if requestJson["minre"] != "" || requestJson["maxre"] != "" {
		message = "The \"minre\" and \"maxre\" parameters are no longer supported. Use \"video\" instead."
		Info.Println(message)
		warningMessages = append(warningMessages, message)
	}

	dtls := requestJson["dtls"]
	dscp := requestJson["dscp"]
	ipv6 := requestJson["ipv6"]

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
		roomLink := request.Host + "/r/" + roomid + "?" + request.URL.Query().Encode()
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

func getWssParameters(requestJson map[string]string) (wssUrl string, wssPostUrl string) {
	wssHostPortPair := requestJson["wshpp"]
	wssTls := requestJson["wstls"]
	if wssHostPortPair == "" {
		wssHostPortPair = WSS_INSTANCES[0][WSS_INSTANCE_HOST_KEY]
	}

	if wssTls == "false" {
		wssUrl = fmt.Sprintf("ws://%s/ws", wssHostPortPair)
		wssPostUrl = fmt.Sprintf("http://%s", wssHostPortPair)
	} else {
		wssUrl = fmt.Sprintf("wss://%s/ws", wssHostPortPair)
		wssPostUrl = fmt.Sprintf("https://%s", wssHostPortPair)
	}
	return
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
