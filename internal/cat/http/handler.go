package http

import (
	"github.com/gin-gonic/gin"
	"io/ioutil"
)

func (s *Server) pushKeys(c *gin.Context) {

	var args struct {
		Op   int32    `form:"op" binding:"required"`
		Keys []string `form:"keys" binding:"required"`
	}

	if err := c.BindQuery(&args); err != nil {
		Error(c, ErrRequest, err.Error())
		return
	}

	msg, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		Error(c, ErrRequest, err.Error())
		return
	}

	if err = s.cat.PutKeys(args.Op, args.Keys, msg); err != nil {
		Error(c, ErrRequest, err.Error())
	}

	Success(c, nil, OK)
}
