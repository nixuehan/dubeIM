package http

import (
	"dube/internal/cat"
	"dube/internal/cat/options"
	"github.com/gin-gonic/gin"
)

type Server struct {
	engine *gin.Engine
	cat    *cat.Cat
}

func NewServer(engine *gin.Engine, cat *cat.Cat) *Server {
	s := &Server{}
	s.engine = engine
	s.cat = cat
	s.initRouter()
	return s
}

func New(c *options.HTTPServer, cat *cat.Cat) *Server {
	engine := gin.New()
	engine.Use(loggerHandler, recoverHandler)
	go func() {
		if err := engine.Run(c.Addr); err != nil {
			panic(err)
		}
	}()

	return NewServer(engine, cat)
}

//curl -X POST http://127.0.0.1:3111/dubeim/push/key
func (s *Server) initRouter() {
	g := s.engine.Group("/dubeim")
	g.POST("/push/keys", s.pushKeys)
}

func (s *Server) GracefulStop() {

}
