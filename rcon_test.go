package rcon_test

import (
	"bytes"
	"encoding/binary"
	"github.com/james4k/rcon"
	"net"
	"testing"
)

func startTestServer() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		buf := make([]byte, 256)
		_, err = conn.Read(buf)
		if err != nil {
			return
		}

		var packetSize, requestId, cmdType int32
		var str []byte
		b := bytes.NewBuffer(buf)
		binary.Read(b, binary.LittleEndian, &packetSize)
		binary.Read(b, binary.LittleEndian, &requestId)
		binary.Read(b, binary.LittleEndian, &cmdType)
		str, err = b.ReadBytes(0x00)
		if err != nil {
			return
		}
		if string(str[:len(str)-1]) != "blerg" {
			requestId = -1
		}

		b.Reset()
		binary.Write(b, binary.LittleEndian, int32(10))
		binary.Write(b, binary.LittleEndian, int32(requestId))
		binary.Write(b, binary.LittleEndian, int32(2))
		binary.Write(b, binary.LittleEndian, byte(0))
		binary.Write(b, binary.LittleEndian, byte(0))
		conn.Write(b.Bytes())
	}()

	return listener.Addr().String(), nil
}

func TestAuth(t *testing.T) {
	addr, err := startTestServer()
	if err != nil {
		t.Fatal(err)
	}

	rc, err := rcon.New(addr, "blerg")
	if err != nil {
		t.Fatal(err)
	}

	err = rc.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// TODO: test multipacket responses
