package main

import (
	"earthworm/socket"
	"encoding/binary"
	"flag"
	"io"
	"net"
	"time"
)

func main() {
	password := flag.String("p", "", "password")
	server := flag.String("h", "", "server")
	flag.Parse()
	if len(*password) == 0 || len(*server) == 0 {
		flag.Usage()
		return
	}
	for {
		time.Sleep(5 * time.Second)
		connectServer(*server, *password)
	}
}

func connectServer(address string, password string) {
	tmpConn, err := net.DialTimeout("tcp", address, 12*time.Second)
	if err != nil {
		return
	}
	conn := socket.New(tmpConn)
	defer conn.Close()
	conn.Write([]byte(password))
	pclient := NewClient(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		var head [5]byte
		if _, err := io.ReadFull(conn, head[:]); err != nil {
			break
		}
		bodyLen := binary.LittleEndian.Uint16(head[:])
		sid := binary.LittleEndian.Uint16(head[2:])
		op := head[4]
		bodyData := make([]byte, bodyLen)
		if bodyLen != 0 {
			if _, err := io.ReadFull(conn, bodyData); err != nil {
				break
			}
		}
		pclient.ServerMsg(int(sid), op, bodyData)
	}
	pclient.Stop()
}
