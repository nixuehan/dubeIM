package main

import (
	"dube/internal/cat"
	"dube/internal/cat/http"
	"dube/internal/cat/options"
	"dube/internal/cat/rpc"
	"dube/pkg/program"
	log "github.com/golang/glog"
	"google.golang.org/grpc"
)

type app struct {
	options *options.Options
	grpcSrv *grpc.Server
	httpSrv *http.Server
}

func (c *app) Init() {
	var err error
	c.options, err = options.InitOptions()
	if err != nil {
		log.Errorf("fail to init conf - (%v)", err)
	}
}

func (c *app) Start() {
	s := cat.New(c.options)
	c.grpcSrv = rpc.New(c.options.RpcServer, s)
	c.httpSrv = http.New(c.options.HTTPServer, s)
}

func (c *app) Stop() {
	c.grpcSrv.GracefulStop()
	c.httpSrv.GracefulStop()
}

func main() {
	program := new(program.Program)
	program.Run(new(app))
}
