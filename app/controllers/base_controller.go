package controllers

import (
	"net/http"

	"github.com/beego/beego/v2/server/web"
)

// BaseController provides helpers for consistent JSON responses.
type BaseController struct {
	web.Controller
}

// JSON writes a JSON response with the supplied HTTP status code.
func (c *BaseController) JSON(status int, payload interface{}) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = payload
	c.ServeJSON()
}

// JSONSuccess writes a standard success envelope.
func (c *BaseController) JSONSuccess(data interface{}) {
	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

// JSONError writes an error envelope with message.
func (c *BaseController) JSONError(status int, message string) {
	c.JSON(status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}
