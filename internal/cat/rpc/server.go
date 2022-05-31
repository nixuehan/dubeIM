package rpc

import (
	"context"
	"dube/internal/cat"
	"dube/internal/cat/options"
	pb "dube/internal/protocol/cat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"net"
	"time"
)

type Server struct {
	srv *cat.Cat
}

func New(c *options.RpcServer, cat *cat.Cat) *grpc.Server {
	opt := grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle:     time.Duration(c.IdleTimeout),
		MaxConnectionAge:      time.Duration(c.MaxLifeTime),
		MaxConnectionAgeGrace: time.Duration(c.ForceCloseWait),
		Timeout:               time.Duration(c.KeepAliveTimeout),
		Time:                  time.Duration(c.KeepAliveInterval),
	})

	srv := grpc.NewServer(opt)
	pb.RegisterCatServer(srv, &Server{cat})

	l, err := net.Listen(c.Network, c.Addr)
	if err != nil {
		panic(err)
	}

	go func() {
		if err = srv.Serve(l); err != nil {
			panic(err)
		}
	}()

	return srv
}

func (s *Server) Identify(ctx context.Context, req *pb.IdentifyReq) (*pb.IdentifyResp, error) {
	mid, key, roomID, hb, err := s.srv.Identify(ctx, req.GetServer(), req.GetToken())
	if err != nil {
		return nil, err
	}
	resp := new(pb.IdentifyResp)
	resp.Mid = mid
	resp.Key = key
	resp.RoomID = roomID
	resp.Heartbeat = hb
	return resp, nil
}

func (s *Server) Heartbeat(ctx context.Context, req *pb.HeartbeatReq) (*pb.HeartbeatResp, error) {
	if err := s.srv.Heartbeat(ctx, req.GetMid(), req.GetKey(), req.GetServer()); err != nil {
		return nil, err
	}
	return &pb.HeartbeatResp{}, nil
}
