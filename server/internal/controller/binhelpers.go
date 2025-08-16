package controller

import (
	"time"

	binproto "myflowhub/pkg/protocol/binproto"
	"myflowhub/server/internal/hub"

	"github.com/gorilla/websocket"
)

func sendFrame(s *hub.Server, c *hub.Client, h binproto.HeaderV1, typeID uint16, payload []byte) {
	frame, _ := binproto.EncodeFrame(binproto.HeaderV1{TypeID: typeID, MsgID: h.MsgID, Source: s.DeviceID, Target: c.DeviceID, Timestamp: time.Now().UnixMilli()}, payload)
	_ = c.Conn.WriteMessage(websocket.BinaryMessage, frame)
}

func sendOK(s *hub.Server, c *hub.Client, h binproto.HeaderV1, code int32, msg string) {
	pl := binproto.EncodeOKResp(h.MsgID, code, []byte(msg))
	sendFrame(s, c, h, binproto.TypeOKResp, pl)
}

func sendErr(s *hub.Server, c *hub.Client, h binproto.HeaderV1, code int32, msg string) {
	pl := binproto.EncodeErrResp(h.MsgID, code, []byte(msg))
	sendFrame(s, c, h, binproto.TypeErrResp, pl)
}
