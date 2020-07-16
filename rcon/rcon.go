package rcon

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	cmdAuth        = 3
	cmdExecCommand = 2

	respResponse     = 0
	respAuthResponse = 2
)

// 12 byte header, up to 4096 bytes of data, 2 bytes for null terminators.
// this should be the absolute max size of a single response.
const readBufferSize = 4110

type RemoteConsole struct {
	conn      net.Conn
	readbuf   []byte
	readmu    sync.Mutex
	reqid     int32
	queuedbuf []byte
}

var (
	ErrAuthFailed          = errors.New("rcon: authentication failed")
	ErrInvalidAuthResponse = errors.New("rcon: invalid response type during auth")
	ErrUnexpectedFormat    = errors.New("rcon: unexpected response format")
	ErrCommandTooLong      = errors.New("rcon: command too long")
	ErrResponseTooLong     = errors.New("rcon: response too long")
)

func Dial(ctx context.Context, host, password string, timeout time.Duration) (*RemoteConsole, error) {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, err
	}
	var reqid int
	r := &RemoteConsole{conn: conn, reqid: 0x7fffffff}
	reqid, err = r.writeCmd(cmdAuth, password)
	if err != nil {
		return nil, err
	}

	r.readbuf = make([]byte, readBufferSize)

	var respType, requestId int
	respType, requestId, _, err = r.readResponse(timeout)
	if err != nil {
		return nil, err
	}

	// if we didn't get an auth response back, try again. it is often a bug
	// with RCON servers that you get an empty response before receiving the
	// auth response.
	if respType != respAuthResponse {
		respType, requestId, _, err = r.readResponse(timeout)
	}
	if err != nil {
		return nil, err
	}
	if respType != respAuthResponse {
		return nil, ErrInvalidAuthResponse
	}
	if requestId != reqid {
		return nil, ErrAuthFailed
	}

	return r, nil
}

func (r *RemoteConsole) LocalAddr() net.Addr {
	return r.conn.LocalAddr()
}

func (r *RemoteConsole) RemoteAddr() net.Addr {
	return r.conn.RemoteAddr()
}

func (r *RemoteConsole) Write(cmd string) (requestId int, err error) {
	return r.writeCmd(cmdExecCommand, cmd)
}

func (r *RemoteConsole) Read() (response string, requestId int, err error) {
	var respType int
	var respBytes []byte
	respType, requestId, respBytes, err = r.readResponse(2 * time.Minute)
	if err != nil || respType != respResponse {
		response = ""
		requestId = 0
	} else {
		response = string(respBytes)
	}
	return
}

func (r *RemoteConsole) Close() error {
	return r.conn.Close()
}

func newRequestId(id int32) int32 {
	if id&0x0fffffff != id {
		return int32((time.Now().UnixNano() / 100000) % 100000)
	}
	return id + 1
}

func (r *RemoteConsole) writeCmd(cmdType int32, str string) (int, error) {
	if len(str) > 1024-10 {
		return -1, ErrCommandTooLong
	}

	buffer := bytes.NewBuffer(make([]byte, 0, 14+len(str)))
	reqid := atomic.LoadInt32(&r.reqid)
	reqid = newRequestId(reqid)
	atomic.StoreInt32(&r.reqid, reqid)

	// packet size
	binary.Write(buffer, binary.LittleEndian, int32(10+len(str)))

	// request id
	binary.Write(buffer, binary.LittleEndian, int32(reqid))

	// auth cmd
	binary.Write(buffer, binary.LittleEndian, int32(cmdType))

	// string (null terminated)
	buffer.WriteString(str)
	binary.Write(buffer, binary.LittleEndian, byte(0))

	// string 2 (null terminated)
	// we don't have a use for string 2
	binary.Write(buffer, binary.LittleEndian, byte(0))

	r.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := r.conn.Write(buffer.Bytes())
	return int(reqid), err
}

func (r *RemoteConsole) readResponse(timeout time.Duration) (int, int, []byte, error) {
	r.readmu.Lock()
	defer r.readmu.Unlock()

	r.conn.SetReadDeadline(time.Now().Add(timeout))
	var size int
	var err error
	if r.queuedbuf != nil {
		copy(r.readbuf, r.queuedbuf)
		size = len(r.queuedbuf)
		r.queuedbuf = nil
	} else {
		size, err = r.conn.Read(r.readbuf)
		if err != nil {
			return 0, 0, nil, err
		}
	}
	if size < 4 {
		// need the 4 byte packet size...
		s, err := r.conn.Read(r.readbuf[size:])
		if err != nil {
			return 0, 0, nil, err
		}
		size += s
	}

	var dataSize32 int32
	b := bytes.NewBuffer(r.readbuf[:size])
	binary.Read(b, binary.LittleEndian, &dataSize32)
	if dataSize32 < 10 {
		return 0, 0, nil, ErrUnexpectedFormat
	}

	totalSize := size
	dataSize := int(dataSize32)
	if dataSize > 4106 {
		return 0, 0, nil, ErrResponseTooLong
	}

	for dataSize+4 > totalSize {
		size, err := r.conn.Read(r.readbuf[totalSize:])
		if err != nil {
			return 0, 0, nil, err
		}
		totalSize += size
	}

	data := r.readbuf[4 : 4+dataSize]
	if totalSize > dataSize+4 {
		// start of the next buffer was at the end of this packet.
		// save it for the next read.
		r.queuedbuf = r.readbuf[4+dataSize : totalSize]
	}

	return r.readResponseData(data)
}

func (r *RemoteConsole) readResponseData(data []byte) (int, int, []byte, error) {
	var requestId, responseType int32
	var response []byte
	b := bytes.NewBuffer(data)
	binary.Read(b, binary.LittleEndian, &requestId)
	binary.Read(b, binary.LittleEndian, &responseType)
	response, err := b.ReadBytes(0x00)
	if err != nil && err != io.EOF {
		return 0, 0, nil, err
	}
	if err == nil {
		// if we didn't hit EOF, we have a null byte to remove
		response = response[:len(response)-1]
	}
	return int(responseType), int(requestId), response, nil
}

func (r *RemoteConsole) Exec(c string) (string, error) {
	wid, err := r.Write(c)
	if err != nil {
		log.Fatalf("Failed to write RCON command")
	}
	for {
		resp, rid, err := r.Read()
		if err != nil {
			log.Fatalf("Failed to read response")
		}
		if wid == rid {
			return resp, nil
		}
	}
}
