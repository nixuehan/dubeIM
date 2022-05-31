package http

import "github.com/gin-gonic/gin"

const (
	OK         = 0
	ErrRequest = -400
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Error(c *gin.Context, code int, message string) {
	c.JSON(200, &Response{
		Code:    code,
		Message: message,
	})
}

func Success(c *gin.Context, data interface{}, code int) {
	c.JSON(200, &Response{
		Code: code,
		Data: data,
	})
}
