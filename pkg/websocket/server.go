package websocket

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	log "github.com/golang/glog"
	"net"
	"net/http"
	"strings"
)

var (
	ErrRequestMethod       = errors.New("websocket: bad method")
	ErrSecWebSocketKey     = errors.New("websocket: bad Sec-WebSocket-Accept")
	ErrSecWebsocketVersion = errors.New("websocket: bad Sec-Websocket-Version")
	ErrUpgrade             = errors.New("websocket: bad Upgrade")
	ErrConnection          = errors.New("websocket: bad Connection")
	ErrMessageClose        = errors.New("websocket: close control message")
)

const (
	TextMessage   = 1
	BinaryMessage = 2
	PingMessage   = 9
	PongMessage   = 10
	CloseMessage  = 8
)

type Conn struct {
	Request *Request
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
	buf     []byte
	n       int64
}

type Websocket struct {
	Request *Request
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
}

func New(c net.Conn, r *bufio.Reader, w *bufio.Writer) *Websocket {
	ws := &Websocket{}
	ws.reader = r
	ws.writer = w
	ws.conn = c
	req, err := ws.handShake()
	if err != nil {
		log.Errorf("fail to handshake for websocket - (%s)", err)
	}

	ws.Request = req

	return ws
}

//https://www.cnblogs.com/yjf512/p/2915171.html?ivk_sa=1024320u
func (w *Websocket) handShake() (*Request, error) {
	r := NewRequest(w.reader)

	l, err := r.ReadLine()
	if err != nil {
		return nil, err
	}

	f := bytes.Fields(l)
	r.Method = strings.ToLower(string(f[0]))
	r.RequestURI = strings.ToLower(string(f[1]))

	header := make(http.Header)
	for {
		l, err = r.ReadLine()
		if err != nil {
			return nil, err
		}

		if l == nil {
			break
		}

		f = bytes.Fields(l)

		header.Add(string(f[0][0:bytes.IndexByte(f[0], ':')]), string(f[1]))
	}

	r.Header = header

	return r, nil
}

func (w *Websocket) Upgrade() error {
	if w.Request.Method != "get" {
		return ErrRequestMethod
	}

	if w.Request.Header.Get("Sec-Websocket-Version") != "13" {
		return ErrSecWebsocketVersion
	}

	if w.Request.Header.Get("Upgrade") != "websocket" {
		return ErrUpgrade
	}

	if strings.ToLower(w.Request.Header.Get("Connection")) != "upgrade" {
		return ErrConnection
	}

	k := w.Request.Header.Get("Sec-WebSocket-Key")

	if k == "" {
		return ErrSecWebSocketKey
	}

	builder := strings.Builder{}
	builder.WriteString("HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n")
	builder.WriteString("Sec-WebSocket-Accept: " + w.calculateKey(k) + "\r\n\r\n")
	s := builder.String()

	_, err := w.writer.WriteString(s)
	if err != nil {
		return err
	}
	if err = w.writer.Flush(); err != nil {
		return err
	}

	return nil
}

func (w *Websocket) calculateKey(k string) string {
	GUID := "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	hash := sha1.New()
	hash.Write([]byte(k))
	hash.Write([]byte(GUID))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func (w *Websocket) Close() {
	w.conn.Close()
}

func NewConn(w *Websocket) *Conn {
	return &Conn{
		w.Request,
		w.conn,
		w.reader,
		w.writer,
		make([]byte, 1024),
		0,
	}
}

func (c *Conn) Peek(n int64) []byte {
	b := c.buf[c.n : c.n+n]
	c.n += n
	return b
}

func (c *Conn) Buffer() []byte {
	defer func() {
		c.buf = c.buf[:0]
	}()
	return c.buf[:c.n]
}

func (c *Conn) Header(op int, payloadLen uint64) {
	b := c.Peek(2)

	b[0] = 1
	b[0] = b[0]<<7 | byte(op)
	b[1] = 0

	switch {
	case payloadLen <= 125:
		b[1] |= byte(payloadLen)
	case payloadLen <= 65535:
		b[1] |= 126
		b = c.Peek(2)
		binary.BigEndian.PutUint16(b, uint16(payloadLen))
	default:
		b[1] |= 127
		b = c.Peek(8)
		binary.BigEndian.PutUint64(b, payloadLen)
	}
}

func (c *Conn) WriteBody(b []byte) error {
	//copy(c.buf[c.n:], b)
	if len(b) > 0 {
		_, err := c.writer.Write(b)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Conn) Flush() {
	c.writer.Flush()
}

func (c *Conn) ReadFrame() (fin bool, op int, payload []byte, err error) {

	b := make([]byte, 2)
	_, err = c.reader.Read(b)
	if err != nil {
		return
	}

	fin = b[0]>>7 != 0

	if rsv := b[0] & 0x70; rsv != 0 {
		err = ErrConnection
		return
	}

	op = int(b[0] & 0xF)
	mask := b[1]>>7 != 0

	var (
		payloadLen uint64
		maskKey    []byte
	)

	switch int8(b[1] & 0x7F) {
	case 126:
		b = make([]byte, 2)
		_, err = c.reader.Read(b)
		if err != nil {
			return
		}
		payloadLen = binary.BigEndian.Uint64(b)
	case 127:
		b = make([]byte, 8)
		_, err = c.reader.Read(b)
		if err != nil {
			return
		}
		payloadLen = binary.BigEndian.Uint64(b)
	default:
		payloadLen = uint64(b[1] & 0x7F)
	}

	if mask {
		maskKey = make([]byte, 4)
		_, err = c.reader.Read(maskKey)
		if err != nil {
			return
		}
	}

	if payloadLen > 0 {
		payload = make([]byte, payloadLen)
		_, err = c.reader.Read(payload)
		if err != nil {
			return
		}
	}

	if maskKey != nil {
		var i uint64
		for i = 0; i < payloadLen; i++ {
			payload[i] = payload[i] ^ maskKey[i%4]
		}
	}

	return
}

func (c *Conn) Write(op int, b []byte) error {
	//buf := make([]byte, 2*1024)
	//c.Header(op, uint64(len(b)), buf)
	//copy(buf[len(buf):], b)
	//if err := c.WriteBody(buf); err != nil {
	//	return err
	//}
	return nil
}

func (c *Conn) Read() (op int, payload []byte, err error) {
	var (
		fin bool
		p   []byte
	)
	for {
		if fin, op, p, err = c.ReadFrame(); err != nil {
			return
		}

		switch op {
		case CloseMessage:
			err = ErrMessageClose
			return
		case PingMessage:
			if err = c.Write(PongMessage, p); err != nil {
				log.Errorf("fail to return pong - (%s)", err)
				return
			}
		case BinaryMessage, TextMessage:
			if fin && len(p) == 0 {
				return
			}
			payload = append(payload, p...)

			if fin {
				return
			}
		default:
			err = fmt.Errorf("unknown protocol - (fin=%t op=%d)", fin, op)
			return
		}
	}
}

func (c *Conn) Close() {
	c.conn.Close()
}
