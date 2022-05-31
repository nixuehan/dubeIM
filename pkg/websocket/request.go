package websocket

import (
	"bufio"
	"net/http"
)

type Request struct {
	Method     string
	RequestURI string
	Header     http.Header
	reader     *bufio.Reader
}

func NewRequest(r *bufio.Reader) *Request {
	req := &Request{}
	req.reader = r
	return req
}

func (r *Request) ReadLine() ([]byte, error) {

	var (
		line []byte
	)

	for {
		l, more, err := r.reader.ReadLine()
		if err != nil {
			return l, err
		}

		if l == nil && !more {
			return l, nil
		}

		line = append(line, l...)

		if !more {
			return line, nil
		}
	}
}
