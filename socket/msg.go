package socket

import (
	"encoding/binary"
	"fmt"
)

const (
	HEARTBEAT = 0
	CONNECT   = 1
	CLOSE     = 2
	DATA      = 3
)

func MakeMsg(sid uint16, op byte, body []byte) []byte {
	length := len(body)
	if length > 65535 {
		panic(fmt.Sprintf("body length %d > 65535", length))
	}
	pmsg := make([]byte, 5+length)
	binary.LittleEndian.PutUint16(pmsg, uint16(length))
	binary.LittleEndian.PutUint16(pmsg[2:], sid)
	pmsg[4] = op
	if length != 0 {
		copy(pmsg[5:], body)
	}
	return pmsg
}
