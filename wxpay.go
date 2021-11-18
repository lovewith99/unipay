package unipay

import (
	"errors"

	wxpayv2 "github.com/lovewith99/wxpay/v2"
)

type WxPayClient struct {
	WxPayClientConfig

	client    *wxpayv2.Client
	OrderSvc  UniPayOrderService
	OrderInfo func(UniPayOrder) *UniPayOrderInfo
}

func (cli *WxPayClient) Client() *wxpayv2.Client {
	return cli.client
}

func (cli *WxPayClient) Payment(ctx *Context) (MapResult, error) {
	svc := cli.OrderSvc

	order, err := svc.PostOrder(ctx)
	if err != nil {
		return nil, err
	}

	obj := wxpayv2.UnifiedOrder{}
	obj.TradeType = wxpayv2.APP
	obj.SpbillCreateIp = ctx.ClientIP
	obj.NotifyUrl = cli.NotifyURL

	if cli.signType != "" {
		obj.SignType = cli.signType
	} else {
		obj.SignType = wxpayv2.MD5
	}

	info := cli.OrderInfo(order)
	obj.Body = info.Subject
	obj.OutTradeNo = info.OutTradeNo
	obj.TotalFee = info.TotalFee
	obj.Attach = info.Attach

	var resp wxpayv2.UnifiedOrderResp
	if err := cli.client.Do(&obj, &resp); err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, errors.New(resp.ReturnMsg)
	}

	data := resp.RequestData(cli.client)
	return data, nil
}

type WxPayClientOption func(*WxPayClient)

func NewWxPayClient(opts ...WxPayClientOption) (*WxPayClient, error) {
	var err error
	cli := &WxPayClient{}

	for _, opt := range opts {
		opt(cli)
	}

	wxpayopts := make([]func(*wxpayv2.Client) error, 0)
	if cli.certPem != "" && cli.keyPem != "" {
		wxpayopts = append(wxpayopts,
			wxpayv2.WithTlsFile(cli.certPem, cli.keyPem))
	}

	cli.client, err = wxpayv2.NewWxPay(cli.appId, cli.mchId, cli.key, wxpayopts...)

	if err != nil {
		return nil, err
	}

	return cli, err
}

func WxPayConfig(appId, mchId, key string) WxPayClientOption {
	return func(cli *WxPayClient) {
		cli.appId = appId
		cli.mchId = mchId
		cli.key = key
	}
}

func WxPayTLS(certPem, keyPem string) WxPayClientOption {
	return func(cli *WxPayClient) {
		cli.certPem = certPem
		cli.keyPem = keyPem
	}
}

func WxPayNotifyURL(uri string) WxPayClientOption {
	return func(cli *WxPayClient) {
		cli.NotifyURL = uri
	}
}

func WxPayOrderSvc(svc UniPayOrderService) WxPayClientOption {
	return func(cli *WxPayClient) {
		cli.OrderSvc = svc
	}
}

func WxPayGetOrderInfoFunc(f func(UniPayOrder) *UniPayOrderInfo) WxPayClientOption {
	return func(cli *WxPayClient) {
		cli.OrderInfo = f
	}
}
