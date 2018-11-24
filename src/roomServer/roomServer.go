package roomServer

import (
	"net/http"
	"github.com/gorilla/mux"
	"fmt"
	"strconv"
	"encoding/json"
	"strings"
	. "common"
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
	Message     []string `json:"Message"`
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
	router.HandleFunc("/test/{roomid}", rs.test).Methods("POST")
	router.HandleFunc("/join/{roomid}", rs.joinRoomHandler).Methods("POST")
	router.HandleFunc("/message/{roomid}/{clientid}", rs.messageRoomHandler).Methods("POST")

	http.ListenAndServe("0.0.0.0:"+strconv.Itoa(p), router)
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

func GetRequestJson(r *http.Request) (map[string]string,string) {
	body, _ := ioutil.ReadAll(r.Body)
	requestJson := make(map[string]string)
	r.Body.Close()
	if err := json.Unmarshal(body, &requestJson); err == nil {
		Info.Println(requestJson)
	} else {
		Error.Println(err)
	}
	//todo body's encode
	return requestJson,string(body)
}
