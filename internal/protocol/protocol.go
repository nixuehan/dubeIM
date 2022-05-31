package protocol

import (
	"context"
	"dube/internal/protocol/cat"
	"dube/internal/scratcher"
	"dube/pkg/websocket"
	"encoding/binary"
	"errors"
)

const (
	OpHeartbeat = 1
	OpAuthReply = 8
)

const (
	maxPackLen = int32(1 << 12)
)

const (
	_packSize      = 4
	_verSize       = 2
	_operationSize = 4
	_seqSize       = 4
	_rawHeaderSize = _packSize + _verSize + _operationSize + _seqSize

	//offset
	_packOffset      = 0
	_verOffset       = 4
	_operationOffset = 6
	_seqOffset       = 10
)

var (
	ErrPackLen = errors.New("proto: proto pack length error")
)

type Protocol struct {
	*Proto
	s *scratcher.Scratcher
}

func NewProtocol(s *scratcher.Scratcher) *Protocol {
	return &Protocol{&Proto{}, s}
}

func (p *Protocol) Exec() error {
	return nil
}

func (p *Protocol) OpHeartbeat(mid int64, key, server string) error {
	_, err := p.s.RpcClient.Heartbeat(context.Background(), &cat.HeartbeatReq{Mid: mid, Key: key, Server: server})
	if err != nil {
		return err
	}
	return nil
}

func (p *Protocol) ReadWebsocket(conn *websocket.Conn) error {

	var (
		err  error
		data []byte
	)
	if _, data, err = conn.Read(); err != nil {
		return err
	}

	if _rawHeaderSize >= len(data) {
		return ErrPackLen
	}

	packageLen := int32(binary.BigEndian.Uint32(data[_packOffset:_verOffset]))
	if packageLen >= maxPackLen {
		return ErrPackLen
	}

	p.Ver = int32(binary.BigEndian.Uint16(data[_verOffset:_operationOffset]))
	p.Op = int32(binary.BigEndian.Uint32(data[_operationOffset:_seqOffset]))
	p.Seq = int32(binary.BigEndian.Uint32(data[_seqOffset:_rawHeaderSize]))
	p.Body = data[_rawHeaderSize:packageLen]
	return nil
}

func (p *Protocol) WriteWebsocket(conn *websocket.Conn) error {
	payloadLen := _rawHeaderSize + len(p.GetBody())

	conn.Header(websocket.BinaryMessage, uint64(payloadLen))

	h := conn.Peek(_rawHeaderSize)

	binary.BigEndian.PutUint32(h[_packOffset:], uint32(payloadLen))
	binary.BigEndian.PutUint16(h[_verOffset:], uint16(p.GetVer()))
	binary.BigEndian.PutUint32(h[_operationOffset:], uint32(p.GetOp()))
	binary.BigEndian.PutUint32(h[_seqOffset:], uint32(p.Seq))

	if err := conn.WriteBody(conn.Buffer()); err != nil {
		return err
	}

	if err := conn.WriteBody(p.GetBody()); err != nil {
		return err
	}

	conn.Flush()

	return nil
}
