package otime

import (
	"fmt"
	"testing"
)

func TestOtime(t *testing.T) {
	var a Duration
	a.UnmarshalText([]byte("1s"))
	fmt.Printf("%d", int64(a))
}
