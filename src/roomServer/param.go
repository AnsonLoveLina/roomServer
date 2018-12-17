package roomServer

import (
	"net/http"

	. "common"
)

func (rs *RoomServer) paramRoomHandler(rw http.ResponseWriter, r *http.Request) {
	requestJson, _ := GetRequestJson(r)
	params := getRoomParameters(r, requestJson, "", "", nil)
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Write(MarshalNoError(params, []byte("")))
}
