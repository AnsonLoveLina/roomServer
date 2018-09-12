package roomServer

import (
	"net/http"
	"github.com/gorilla/mux"
	"math/rand"
	"fmt"
	"strconv"
	"log"
	"encoding/json"
	"strings"
	. "../common"
	"net/url"
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
		return "room is full"
	case duplicateClient:
		return "duplicate client"
	default:
		return "unknown"
	}
}

type Room struct {
	Clients map[int]*Client `json:"clients"`
}

type Client struct {
	IsInitiator bool   `json:"IsInitiator"`
	Message     string `json:"Message"`
}

func NewClient(isInitiator bool) *Client {
	return &Client{IsInitiator: isInitiator}
}

func NewRoom() *Room {
	return &Room{Clients: make(map[int]*Client)}
}

type RoomServer struct {
	RedisClient *RedisClient
}

func NewRoomServer(redisClient *RedisClient) *RoomServer{
	roomServer := &RoomServer{redisClient}
	return roomServer
}

func (rs *RoomServer) Run(p int, tls bool) {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/join/{roomid+}", rs.joinRoomHandler).Methods("POST")

	http.ListenAndServe("0.0.0.0:"+strconv.Itoa(p), router)
}

func (rs *RoomServer) joinRoomHandler(rw http.ResponseWriter, r *http.Request) {
	roomid := mux.Vars(r)["roomid"]
	clientid := rand.Int()

	result := rs.addClient2Room(r, roomid, clientid)
	if result.error != "" {
		log.Println("Error adding client to room: %s, room_state=%s",
			result.error, result.room)
		writeResponse(rw, result.error, make(map[string]interface{}), make([]string, 0))
		return
	}

	params := getRoomParameters(r, roomid, clientid, result.isInitiator)
	writeResponse(rw, "SUCCESS", params, result.messages)
}

type result struct {
	error       string   `json:"error"`
	isInitiator bool     `json:"IsInitiator"`
	messages    []string `json:"messages"`
	room        Room     `json:"room_state"`
}

func (rs *RoomServer) addClient2Room(request *http.Request, roomid string, clientid int) (result) {
	var isInitiator = false
	var messages = make([]string, roomMaxOccupancy)
	var error = ""
	//roomKey := fmt.Sprintf("%s/%s", request.URL.Host, roomid)
	var room Room
	var occupancy int
	var roomValue interface{}
	var redisCon = rs.RedisClient.getRedisConnNotNil()

	roomValue, _ = redisCon.Do("HGET", roomid, clientid)
	redisCon.Do("WHATCH", roomid)
	_, err := redisCon.Do("MULTI")
	if err != nil {
		error = redisError.getString()
		return result{error: error}
	}
	if roomValue == nil {
		room = *NewRoom()
		_, err := redisCon.Do("HMSET", roomid, clientid, room)
		if err != nil {
			error = redisError.getString()
			return result{error: error}
		}
	} else {
		roomValue = roomValue.([]string)
	}
	occupancy = len(room.Clients)

	if occupancy >= roomMaxOccupancy {
		error = roomFull.getString()
		return result{error: error}
	}
	if room.Clients[clientid] != nil {
		error = duplicateClient.getString()
		return result{error: error}
	}

	if occupancy == 0 { //the first client of this room
		isInitiator = true
		room.Clients[clientid] = NewClient(isInitiator)
	} else {
		isInitiator = false
		otherClients := room.Clients
		var i = 0
		for _, client := range otherClients {
			messages[i] = client.Message
			i++
			client.Message = ""
		}
		room.Clients[clientid] = NewClient(isInitiator)
	}

	var _, er = redisCon.Do("EXEC")
	if er != nil {
		error = redisError.getString()
		return result{error: error}
	}

	return result{error, isInitiator, messages, room}
}

func writeResponse(rw http.ResponseWriter, result string, params map[string]interface{}, messages []string) {
	params["messages"] = messages
	responseObj := map[string]interface{}{"result": result, "params": params}
	response, error := json.Marshal(responseObj)
	if error != nil {
		err := fmt.Sprintf("json marshal error %s result:%s,params:%s,messages:%s", error.Error(), result, params, messages)
		log.Println(err)
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

func getRoomParameters(request *http.Request, roomid string, clientid int, isInitiator interface{}) (map[string]interface{}) {
	var warningMessages = make([]string, 10)
	var message string
	userAgent := request.UserAgent()
	request.ParseForm()
	iceTransports := request.Form.Get("it")
	iceServerTransports := request.Form.Get("tt")
	var iceServerBaseUrl string
	if request.Form.Get("ts") != "" {
		iceServerBaseUrl = request.Form.Get("ts")
	} else {
		iceServerBaseUrl = ICE_SERVER_BASE_URL
	}

	audio := request.Form.Get("audio")
	video := request.Form.Get("video")

	firefoxFakeDevice := request.Form.Get("firefox_fake_device")

	hd := strings.ToLower(request.Form.Get("hd"))

	if hd != "" && video != "" {
		message = "The \"hd\" parameter has overridden video=" + video
		log.Println(message)
		warningMessages = append(warningMessages, message)
	}
	if hd == "true" {
		video = "mandatory:minWidth=1280,mandatory:minHeight=720"
	} else if hd != "" && video != "" && get_hd_default(userAgent) {
		video = "optional:minWidth=1280,optional:minHeight=720"
	}

	if request.Form.Get("minre") != "" || request.Form.Get("maxre") != "" {
		message = "The \"minre\" and \"maxre\" parameters are no longer supported. Use \"video\" instead."
		log.Println(message)
		warningMessages = append(warningMessages, message)
	}

	dtls := request.Form.Get("dtls")
	dscp := request.Form.Get("dscp")
	ipv6 := request.Form.Get("ipv6")

	var iceServerUrl = ""
	if len(iceServerBaseUrl) > 0 {
		iceServerUrl = fmt.Sprintf(ICE_SERVER_URL_TEMP, ICE_SERVER_BASE_URL, ICE_SERVER_API_KEY)
	}

	iceServerOverride := ICE_SERVER_OVERRIDE
	pcConfig := makePcConfig(iceTransports, iceServerOverride)
	pcConstraints := makePcConstraints(dtls, dscp, ipv6)
	offer_options := struct{}{}
	mediaConstraints := makeMediaStreamConstraints(audio, video, firefoxFakeDevice)
	wssUrl, wssPostUrl := getWssParameters(request)

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
		roomLink := request.URL.Host + "/r/" + roomid
		newRoomURL := url.URL{Host: roomLink, RawQuery: request.URL.Query().Encode()}
		roomLink = newRoomURL.Path
		params["room_link"] = roomLink
	}
	if clientid != 0 {
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

func getWssParameters(request *http.Request) (wssUrl string, wssPostUrl string) {
	wssHostPortPair := request.Form.Get("wshpp")
	wssTls := request.Form.Get("wstls")
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
	if constraints != "" || strings.ToLower(constraints) == "true" {
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
				optional[len(optional)] = map[string]interface{}{tokens[0]: tokens[1]}
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
	log.Printf("Applying media constraints: %s", JsonByte(json.Marshal(stream_constraints)))
	return stream_constraints
}

func makePcConstraints(dtls string, dscp string, ipv6 string) interface{} {

	var optionals = make([]map[string]bool, 0)
	dtlsValue, dtlsError := strconv.ParseBool(dtls)
	if dtlsError != nil {
		optionals[len(optionals)] = map[string]bool{"DtlsSrtpKeyAgreement": dtlsValue}
	}
	dscpValue, dscpError := strconv.ParseBool(dscp)
	if dscpError != nil {
		optionals[len(optionals)] = map[string]bool{"googDscp": dscpValue}
	}
	ipv6Value, ipv6Error := strconv.ParseBool(ipv6)
	if ipv6Error != nil {
		optionals[len(optionals)] = map[string]bool{"googIPv6": ipv6Value}
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
