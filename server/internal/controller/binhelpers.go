package controller

import (
	binproto "myflowhub/pkg/protocol/binproto"
	"myflowhub/server/internal/hub"

	"github.com/rs/zerolog/log"
)

// 所有下行发送必须通过单写协程（client.Send -> writePump），避免与 Ping/Pong 并发写
func sendFrame(s *hub.Server, c *hub.Client, h binproto.HeaderV1, typeID uint16, payload []byte) {
	// 记录入队前的队列状态
	qlen, qcap := len(c.Send), cap(c.Send)
	log.Debug().Uint64("msgID", h.MsgID).Uint16("typeID", typeID).Int("queueLen", qlen).Int("queueCap", qcap).Msg("enqueue frame to client")
	s.SendBin(c, typeID, h.MsgID, c.DeviceID, payload)
}

func sendOK(s *hub.Server, c *hub.Client, h binproto.HeaderV1, code int32, msg string) {
	pl := binproto.EncodeOKResp(h.MsgID, code, []byte(msg))
	sendFrame(s, c, h, binproto.TypeOKResp, pl)
}

func sendErr(s *hub.Server, c *hub.Client, h binproto.HeaderV1, code int32, msg string) {
	pl := binproto.EncodeErrResp(h.MsgID, code, []byte(msg))
	sendFrame(s, c, h, binproto.TypeErrResp, pl)
}
