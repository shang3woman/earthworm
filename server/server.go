package main

import (
	"earthworm/socket"
	"sync"
)

type Server struct {
	clientConn *socket.ConnWrap

	mutex    sync.Mutex
	sessions map[int]*socket.ConnWrap
	index    int
	stop     bool
}

func NewServer(conn *socket.ConnWrap) *Server {
	pserver := new(Server)
	pserver.clientConn = conn
	pserver.sessions = make(map[int]*socket.ConnWrap)
	pserver.index = 1
	return pserver
}

func (s *Server) ClientMsg(sid int, op byte, body []byte) {
	if sid == 0 {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	session := s.sessions[sid]
	if session == nil {
		return
	}
	if op == socket.DATA {
		session.Write(body)
	} else if op == socket.CLOSE {
		session.Close()
		delete(s.sessions, sid)
	}
}

func (s *Server) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.stop = true
	for _, v := range s.sessions {
		v.Close()
	}
	s.sessions = make(map[int]*socket.ConnWrap)
}

func (s *Server) CloseClientConn() {
	s.clientConn.Close()
}

func (s *Server) AddSession(socks5 *socket.ConnWrap) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.stop {
		return 0
	}
	if s.index > 65535 {
		s.clientConn.Close()
		return 0
	}
	sid := s.index
	s.index += 1
	s.sessions[sid] = socks5
	return sid
}

func (s *Server) DeleSession(sid int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	session := s.sessions[sid]
	if session == nil {
		return
	}
	delete(s.sessions, sid)
	s.ClientWrite(socket.MakeMsg(uint16(sid), socket.CLOSE, nil))
}

func (s *Server) ClientWrite(data []byte) {
	s.clientConn.Write(data)
}
