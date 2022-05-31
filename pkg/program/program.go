package program

import (
	"os"
	"os/signal"
	"syscall"
)

type App interface {
	Init()
	Start()
	Stop()
}

type Program struct {
	app App
}

func (p *Program) Init() {

}

func (p *Program) Run(a App) {
	p.app = a
	p.app.Init()
	p.app.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-c:
		p.stop()
	}
}

func (p *Program) stop() {
	p.app.Stop()
}
