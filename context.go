package unipay

import "net/url"

type MapResult map[string]interface{}

type Context struct {
	Request

	Uid      interface{} `json:"-"`
	InApp    interface{} `json:"-"`
	ClientIP string      `json:"-"`
	Currency string      `json:"-"`
	Params   url.Values  `json:"-"`
}

func PayContext(payWay uint8) *Context {
	ctx := &Context{}
	ctx.PayWay = payWay
	return ctx
}
