package http

import (
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"net/http/httputil"
	"runtime"
	"time"
)

func loggerHandler(c *gin.Context) {
	start := time.Now()
	path := c.Request.URL.Path
	method := c.Request.Method
	raw := c.Request.URL.RawQuery

	c.Next()

	end := time.Now()

	latency := end.Sub(start)
	code := c.Writer.Status()
	ip := c.ClientIP()

	if raw != "" {
		path = path + "?" + raw
	}

	log.Infof("method: %s, path: %s, code: %s, ip: %s, time: %s", method, path, code, ip, latency/time.Millisecond)
}

func recoverHandler(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 64<<10)
			runtime.Stack(buf, false)
			request, _ := httputil.DumpRequest(c.Request, false)
			log.Errorf("[panic] panic recovered %s\n%s\n%s\n%s\n", time.Now().Format("2022-06-01 09:54:02"), err, string(request), string(buf))
			c.AbortWithStatus(500)
		}
	}()
	c.Next()
}
