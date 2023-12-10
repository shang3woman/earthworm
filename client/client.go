package main

import (
	"earthworm/socket"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

type DstInfo struct {
	session *socket.ConnWrap
	buffer  []byte
}

type Client struct {
	serverConn *socket.ConnWrap

	mutex    sync.Mutex
	sessions map[int]*DstInfo
}

func NewClient(conn *socket.ConnWrap) *Client {
	pclient := new(Client)
	pclient.serverConn = conn
	pclient.sessions = make(map[int]*DstInfo)
	return pclient
}

func (c *Client) Stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, v := range c.sessions {
		if v.session != nil {
			v.session.Close()
		}
	}
	c.sessions = make(map[int]*DstInfo)
}

func (c *Client) ServerWrite(data []byte) {
	c.serverConn.Write(data)
}

func (c *Client) ServerMsg(sid int, op byte, body []byte) {
	if sid == 0 {
		return
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if op == socket.CONNECT {
		c.sessions[sid] = new(DstInfo)
		go connectDst(c, sid, body)
	} else if op == socket.DATA {
		pdst := c.sessions[sid]
		if pdst == nil {
			return
		}
		if pdst.session == nil {
			pdst.buffer = append(pdst.buffer, body...)
		} else {
			pdst.session.Write(body)
		}
	} else if op == socket.CLOSE {
		pdst := c.sessions[sid]
		if pdst == nil {
			return
		}
		if pdst.session != nil {
			pdst.session.Close()
		}
		delete(c.sessions, sid)
	}
}

func (c *Client) AddSession(sid int, conn *socket.ConnWrap) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	session := c.sessions[sid]
	if session == nil {
		return false
	}
	session.session = conn
	if len(session.buffer) != 0 {
		conn.Write(session.buffer)
		session.buffer = nil
	}
	return true
}

func (c *Client) DelSession(sid int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	session := c.sessions[sid]
	if session == nil {
		return
	}
	delete(c.sessions, sid)
	c.ServerWrite(socket.MakeMsg(uint16(sid), socket.CLOSE, nil))
}

func parseDstAddr(body []byte) string {
	if body[0] == 1 {
		dstip := net.IPv4(body[1], body[2], body[3], body[4])
		return fmt.Sprintf("%s:%d", dstip.String(), binary.BigEndian.Uint16(body[5:]))
	}
	return fmt.Sprintf("%s:%d", string(body[2:len(body)-2]), binary.BigEndian.Uint16(body[len(body)-2:]))
}

func connectDst(c *Client, sid int, body []byte) {
	defer c.DelSession(sid)
	addr := parseDstAddr(body)
	tmpConn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return
	}
	conn := socket.New(tmpConn)
	defer conn.Close()
	if !c.AddSession(sid, conn) {
		return
	}
	var buffer [1024]byte
	for {
		n, err := conn.Read(buffer[:])
		if err != nil {
			break
		}
		if n == 0 {
			continue
		}
		datamsg := socket.MakeMsg(uint16(sid), socket.DATA, buffer[:n])
		c.ServerWrite(datamsg)
	}
}
