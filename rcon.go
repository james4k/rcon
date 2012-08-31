package rcon

import (
	"bytes"
	"encoding/binary"
	"errors"
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

type RemoteConsole struct {
	conn    net.Conn
	readbuf []byte
	readmu  sync.Mutex
	reqid   int32
}

var (
	ErrInvalidAuthResponse = errors.New("rcon: invalid response type during auth")
	ErrAuthFailed          = errors.New("rcon: authentication failed")
)

func New(host, password string) (*RemoteConsole, error) {
	const timeout = 10 * time.Second
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return nil, err
	}

	var reqid int
	r := &RemoteConsole{conn: conn, reqid: 0x7fffffff}
	reqid, err = r.writeCmd(cmdAuth, password)
	if err != nil {
		return nil, err
	}

	var respType, requestId int
	r.readbuf = make([]byte, 4096)
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

func (r *RemoteConsole) Write(cmd string) (requestId int, err error) {
	return r.writeCmd(cmdExecCommand, cmd)
}

func (r *RemoteConsole) Read() (response string, requestId int, err error) {
	var respType int
	respType, requestId, response, err = r.readResponse(2 * time.Minute)
	if err != nil || respType != respResponse {
		response = ""
		requestId = 0
	}
	return
}

func (r *RemoteConsole) Close() error {
	return r.Close()
}

func newRequestId(id int32) int32 {
	if id&0x0fffffff != id {
		return int32((time.Now().UnixNano() / 100000) % 100000)
	}
	return id + 1
}

func (r *RemoteConsole) writeCmd(cmdType int32, str string) (int, error) {
	buffer := bytes.NewBuffer(make([]byte, 14+len(str)))
	reqid := newRequestId(r.reqid)

	// packet size
	binary.Write(buffer, binary.LittleEndian, uint32(10+len(str)))

	// request id
	binary.Write(buffer, binary.LittleEndian, reqid)

	// auth cmd
	binary.Write(buffer, binary.LittleEndian, uint32(cmdType))

	// string (null terminated)
	buffer.WriteString(str)
	binary.Write(buffer, binary.LittleEndian, uint8(0))

	// string 2 (null terminated)
	// we don't have a use for string 2
	binary.Write(buffer, binary.LittleEndian, uint8(0))

	r.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err := r.conn.Write(buffer.Bytes())
	atomic.StoreInt32(&r.reqid, reqid)
	return int(reqid), err
}

func (r *RemoteConsole) readResponse(timeout time.Duration) (int, int, string, error) {
	r.readmu.Lock()
	defer r.readmu.Unlock()

	r.conn.SetReadDeadline(time.Now().Add(timeout))
	_, err := r.conn.Read(r.readbuf)
	if err != nil {
		return 0, 0, "", err
	}

	var responseSize, requestId, responseType int32
	b := bytes.NewBuffer(r.readbuf)
	binary.Read(b, binary.LittleEndian, &responseSize)
	binary.Read(b, binary.LittleEndian, &requestId)
	binary.Read(b, binary.LittleEndian, &responseType)
	response, err := b.ReadString(0x00)

	return int(responseType), int(requestId), response, nil
}
