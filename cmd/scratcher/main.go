package main

import (
	"dube/internal/scratcher"
	"dube/internal/scratcher/conf"
	"dube/pkg/program"
	log "github.com/golang/glog"
	"math/rand"
	"time"
)

type app struct {
}

func (a *app) Init() {

}

func (a *app) Start() {
	c, err := conf.Default()
	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(time.Now().UTC().UnixNano())

	srv := scratcher.New(c)
	scratcher.StartWebsocket(srv)
}

func (a *app) Stop() {
	log.Fatal("stop")
}

func main() {
	program := new(program.Program)
	program.Run(new(app))
}
