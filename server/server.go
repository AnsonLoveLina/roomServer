package server

import (
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"strconv"
	"encoding/json"
	"strings"
	. "common"
	"io/ioutil"
	"errors"
	"github.com/garyburd/redigo/redis"
	"github.com/sirupsen/logrus"
)

type ResponseType int

const roomMaxOccupancy = 2

const errorBreakMax = 10

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
	Clients map[string]*Client `json:"clients"`
}

type Client struct {
	IsInitiator bool     `json:"IsInitiator"`
	Message     []string `json:"Message,omitempty"`
}

func ClientMap(result interface{}, err error) (map[string]*Client, error) {
	values, err := redis.Values(result, err)
	if err != nil {
		return nil, err
	}
	if len(values)%2 != 0 {
		return nil, errors.New("ClientMap: StringMap expects even number of values result")
	}
	m := make(map[string]*Client, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, okKey := values[i].([]byte)
		value, okValue := values[i+1].([]byte)
		if !okKey || !okValue {
			return nil, errors.New("ClientMap: ScanMap key not a bulk string value")
		}
		var client Client
		if er := json.Unmarshal(value, &client); er != nil {
			return nil, errors.New(fmt.Sprintf("ClientMap: json can not transform to Client!key:%s json:%s error:%s", string(key), string(value), er))
			//return nil, errors.New(fmt.Sprintf("ClientMap: json can not transform to Client!json error:%s", er))
		} else {
			m[string(key)] = &client
		}
	}
	return m, nil
}

func NewClient(isInitiator bool) *Client {
	return &Client{IsInitiator: isInitiator}
}

func NewRoom() *Room {
	return &Room{Clients: make(map[string]*Client)}
}

type RoomServer struct {
}

func NewRoomServer() *RoomServer {
	roomServer := &RoomServer{}
	return roomServer
}

func (rs *RoomServer) Run(p int, tls bool) {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/test/{roomid}", rs.test).Methods("POST")
	router.HandleFunc("/join/{roomid}", rs.joinRoomHandler).Methods("POST")
	router.HandleFunc("/message/{roomid}/{clientid}", rs.messageRoomHandler).Methods("POST")
	router.HandleFunc("/leave/{roomid}/{clientid}", rs.leaveRoomHandler).Methods("POST")
	router.HandleFunc("/iceconfig", rs.iceConfig).Methods("POST")
	router.HandleFunc("/params", rs.paramRoomHandler).Methods("GET")

	http.ListenAndServe("0.0.0.0:"+strconv.Itoa(p), router)
}

//todo iceservers的分布式
func (rs *RoomServer) iceConfig(rw http.ResponseWriter, request *http.Request) {
	serverKey := request.URL.Query().Get("key")
	logrus.WithFields(logrus.Fields{"serverKey": serverKey}).Info("iceConfig receive the request")
	rw.Write([]byte(DEFAULT_ICESERVERS))
}

func (rs *RoomServer) test(rw http.ResponseWriter, request *http.Request) {
	roomid := mux.Vars(request)["roomid"]
	fmt.Println(strings.Index(strings.ToLower(request.Proto), "https"))
	fmt.Println(request.URL.Scheme)
	roomLink := request.Host + "/r/" + roomid
	if request.URL.Query().Encode() != "" {
		roomLink = roomLink + "?" + request.URL.Query().Encode()
	}
	fmt.Println(roomLink)
}

func getWssParameters(requestJson map[string]interface{}) (wssUrl string, wssPostUrl string) {
	wssHostPortPair := Interface2string(requestJson["wshpp"], "")
	wssTls := Interface2string(requestJson["wstls"], "")
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

func GetRequestJson(r *http.Request) (map[string]interface{}, string) {
	body, _ := ioutil.ReadAll(r.Body)
	if string(body) == "" {
		return nil, ""
	}
	requestJson := make(map[string]interface{})
	r.Body.Close()
	if err := json.Unmarshal(body, &requestJson); err != nil {
		logrus.WithFields(logrus.Fields{"json": string(body), "error": err}).Error("json unmarshal error")
		//} else {
		//	Info.Println(requestJson)
	}
	//todo body's encode
	return requestJson, string(body)
}
