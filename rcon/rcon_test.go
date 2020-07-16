package rcon

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func startTestServer(fn func(net.Conn, *bytes.Buffer)) (string, error) {
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

		buf := make([]byte, readBufferSize)
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
		binary.Write(b, binary.LittleEndian, int32(respAuthResponse))
		binary.Write(b, binary.LittleEndian, byte(0))
		binary.Write(b, binary.LittleEndian, byte(0))
		conn.Write(b.Bytes())

		if fn != nil {
			b.Reset()
			fn(conn, b)
		}
	}()

	return listener.Addr().String(), nil
}

func TestAuth(t *testing.T) {
	addr, err := startTestServer(nil)
	if err != nil {
		t.Fatal(err)
	}

	rc, err := Dial(context.Background(), addr, "blerg", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	err = rc.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestMultipacket(t *testing.T) {
	addr, err := startTestServer(func(c net.Conn, b *bytes.Buffer) {
		// start packet
		// start response
		binary.Write(b, binary.LittleEndian, int32(10+4000))
		binary.Write(b, binary.LittleEndian, int32(123))
		binary.Write(b, binary.LittleEndian, int32(respResponse))
		for i := 0; i < 4000; i += 1 {
			binary.Write(b, binary.LittleEndian, byte(' '))
		}
		binary.Write(b, binary.LittleEndian, byte(0))
		binary.Write(b, binary.LittleEndian, byte(0))
		// end response
		// start response
		binary.Write(b, binary.LittleEndian, int32(10+4000))
		binary.Write(b, binary.LittleEndian, int32(123))
		binary.Write(b, binary.LittleEndian, int32(respResponse))
		for i := 0; i < 2000; i += 1 {
			binary.Write(b, binary.LittleEndian, byte(' '))
		}
		c.Write(b.Bytes())
		// end packet

		// start packet
		b.Reset()
		for i := 0; i < 2000; i += 1 {
			binary.Write(b, binary.LittleEndian, byte(' '))
		}
		binary.Write(b, binary.LittleEndian, byte(0))
		binary.Write(b, binary.LittleEndian, byte(0))
		// end response
		// start response
		binary.Write(b, binary.LittleEndian, int32(10+2000))
		binary.Write(b, binary.LittleEndian, int32(123))
		binary.Write(b, binary.LittleEndian, int32(respResponse))
		for i := 0; i < 2000; i += 1 {
			binary.Write(b, binary.LittleEndian, byte(' '))
		}
		binary.Write(b, binary.LittleEndian, byte(0))
		binary.Write(b, binary.LittleEndian, byte(0))
		// end response
		// start response - size word is split!
		binary.Write(b, binary.LittleEndian, int32(10+2000))
		c.Write(b.Bytes()[:len(b.Bytes())-3])
		// end packet

		b.Reset()
		binary.Write(b, binary.LittleEndian, int32(10+2000))
		binary.Write(b, binary.LittleEndian, int32(123))
		binary.Write(b, binary.LittleEndian, int32(respResponse))
		for i := 0; i < 2000; i += 1 {
			binary.Write(b, binary.LittleEndian, byte(' '))
		}
		binary.Write(b, binary.LittleEndian, byte(0))
		binary.Write(b, binary.LittleEndian, byte(0))
		// end response
		c.Write(b.Bytes()[1:])
		// end packet
	})
	if err != nil {
		t.Fatal(err)
	}

	rc, err := Dial(context.Background(), addr, "blerg", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	str, _, err := rc.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(str) != 4000 {
		t.Fatal("response length not correct")
	}

	str, _, err = rc.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(str) != 4000 {
		t.Fatal("response length not correct")
	}

	str, _, err = rc.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(str) != 2000 {
		t.Fatal("response length not correct")
	}

	str, _, err = rc.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(str) != 2000 {
		t.Fatal("response length not correct")
	}

	err = rc.Close()
	if err != nil {
		t.Fatal(err)
	}
}
