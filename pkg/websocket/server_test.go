package websocket

import (
	"testing"
)

func TestServer(t *testing.T) {
	a := make([]byte, 5)
	//a[0] = 'c'
	b := a[2:]
	b[0] = 'j'
	println(string(a))
}
