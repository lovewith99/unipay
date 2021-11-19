package uniwxpay

import (
	"errors"

	"github.com/lovewith99/unipay"
	wxpayv2 "github.com/lovewith99/wxpay/v2"
)

type Client struct {
	Config

	client       *wxpayv2.Client
	OrderService unipay.OrderService
}

func (cli *Client) Client() *wxpayv2.Client {
	return cli.client
}

func (cli *Client) Payment(ctx *unipay.Context) (unipay.MapResult, error) {
	svc := cli.OrderService

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

	info := order.OrderInfo()
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

type ClientOption func(*Client)

func NewClient(appId, mchdId, key string, opts ...ClientOption) (*Client, error) {
	var err error
	cli := &Client{}
	cli.appId = appId
	cli.mchId = mchdId
	cli.key = key

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

// func BaseConfig(appId, mchId, key string) ClientOption {
// 	return func(cli *Client) {
// 		cli.appId = appId
// 		cli.mchId = mchId
// 		cli.key = key
// 	}
// }

func NotifyURL(uri string) ClientOption {
	return func(cli *Client) {
		cli.NotifyURL = uri
	}
}

func TLSCertFiles(certPem, keyPem string) ClientOption {
	return func(cli *Client) {
		cli.certPem = certPem
		cli.keyPem = keyPem
	}
}

func WithOrderService(svc unipay.OrderService) ClientOption {
	return func(cli *Client) {
		cli.OrderService = svc
	}
}
