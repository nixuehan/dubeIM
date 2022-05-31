package scratcher

import (
	"bufio"
	"context"
	"dube/internal/protocol"
	"dube/pkg/websocket"
	log "github.com/golang/glog"
	"net"
	"runtime"
	"time"
)

type WSServer struct {
	conn   net.Conn
	Reader *bufio.Reader
	Writer *bufio.Writer
	Server *Scratcher
}

func StartWebsocket(s *Scratcher) error {

	srv := &WSServer{}
	srv.Server = s

	for _, addr := range s.Conf.WebSocket.Bind {
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			log.Fatalf("fail to resolveTCPAddr - (%s)", err)
			return err
		}
		listen, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			log.Fatalf("fail to listen - (%s)", err)
			return err
		}
		for i := 0; i < runtime.NumCPU(); i++ {
			go srv.accept(s, listen)
		}
	}

	return nil
}

func (w *WSServer) accept(s *Scratcher, l *net.TCPListener) {
	for {
		conn, err := l.AcceptTCP()
		if err != nil {
			log.Fatalf("fail to accept - (%s)", err)
			return
		}

		err = conn.SetKeepAlive(s.Conf.WebSocket.KeepAlive)
		if err != nil {
			return
		}
		err = conn.SetReadBuffer(s.Conf.WebSocket.ReadBufferSize)
		if err != nil {
			return
		}

		err = conn.SetWriteBuffer(s.Conf.WebSocket.WriteBufferSize)
		if err != nil {
			return
		}

		w.conn = conn
		w.Reader = bufio.NewReader(conn)
		w.Writer = bufio.NewWriter(conn)
		go w.tcp()
	}
}

func (w *WSServer) tcp() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wb := websocket.New(w.conn, w.Reader, w.Writer)

	if wb.Request.Method != "get" || wb.Request.RequestURI != "/sub" {
		wb.Close()
	}

	if err := wb.Upgrade(); err != nil {
		log.Errorf("fail to upgrade - (%s)", err)
		wb.Close()
	}

	conn := websocket.NewConn(wb)
	p := protocol.NewProtocol(w.Server)

	ch := NewChannel()

	//cat 进行认证 权限认证
	var (
		err error
		hb  time.Duration
	)
	ch.mid, ch.key, ch.roomID, _, err = w.Server.Auth(ctx, p, conn)
	ch.q = make(chan string, 8)
	ch.conn = conn

	if err != nil {
		goto failed
	}

	//conn put into bucket
	if err := w.Server.Bucket.put(ch); err != nil {
		goto failed
	}

	hb = w.Server.RandServerHeartbeat()

	go w.Server.Dispatch(ctx, ch)

	for {
		if err := p.ReadWebsocket(conn); err != nil {
			goto failed
		}

		// refresh  session map
		if p.Op == protocol.OpHeartbeat {
			if now := time.Now(); now.Sub(w.Server.LastHeartbeat) > hb {
				if err := p.OpHeartbeat(ch.mid, ch.key, w.Server.ServerID); err == nil {
					w.Server.LastHeartbeat = now
				}
			}
		}

		if err := w.Server.Handle(p); err != nil {
			goto failed
		}
	}

failed:
	conn.Close()
}

func (w *WSServer) Close() {
	w.conn.Close()
}
