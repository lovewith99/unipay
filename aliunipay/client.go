package alipay

import (
	"fmt"

	"github.com/lovewith99/unipay"
	alipayv3 "github.com/smartwalle/alipay/v3"
)

type Client struct {
	Config

	client       *alipayv3.Client
	OrderService unipay.OrderService
}

func (cli *Client) Client() *alipayv3.Client {
	return cli.client
}

func (cli *Client) Payment(ctx *unipay.Context) (unipay.MapResult, error) {
	svc := cli.OrderService

	// order, err := svc.PostOrder(ctx)
	order, err := svc.PostOrder(ctx)
	if err != nil {
		return nil, err
	}

	obj := alipayv3.TradeAppPay{}
	obj.ProductCode = "QUICK_MSECURITY_PAY"
	obj.NotifyURL = cli.NotifyURL

	info := order.OrderInfo()
	obj.Subject = info.Subject
	obj.OutTradeNo = info.OutTradeNo
	obj.TotalAmount = fmt.Sprintf("%.2f", float64(info.TotalFee)/100)
	obj.PassbackParams = info.Attach

	sign, err := cli.client.TradeAppPay(obj)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"pay_info": sign,
	}, nil
}

func (cli *Client) WapPayment(ctx *unipay.Context) (unipay.MapResult, error) {
	svc := cli.OrderService

	order, err := svc.PostOrder(ctx)
	if err != nil {
		return nil, err
	}

	obj := alipayv3.TradeWapPay{}
	obj.ProductCode = "QUICK_WAP_WAY"
	obj.ReturnURL = cli.ReturnURL
	obj.NotifyURL = cli.NotifyURL

	info := order.OrderInfo()
	obj.Subject = info.Subject
	obj.OutTradeNo = info.OutTradeNo
	obj.TotalAmount = fmt.Sprintf("%.2f", float64(info.TotalFee)/100)
	obj.PassbackParams = info.Attach

	payLink, err := cli.client.TradeWapPay(obj)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		// "order_id": params.OrderObj.ID,
		"pay_link": payLink.String(),
	}, nil
}

type ClientOption func(*Client)

func NewClient(appId, partnerId string, opts ...ClientOption) (*Client, error) {
	var err error
	cli := &Client{}
	cli.appId = appId
	cli.partnerId = partnerId

	for _, opt := range opts {
		opt(cli)
	}

	cli.client, err = alipayv3.New(cli.appId, cli.privateKey, cli.IsProd)
	if err != nil {
		return nil, err
	}

	if cli.Mode == CertMode {
		cli.client.LoadAppPublicCertFromFile(cli.appCertSnFile)
		cli.client.LoadAliPayRootCertFromFile(cli.rootCertSnFile)
		cli.client.LoadAliPayPublicCertFromFile(cli.aliPublicCertSnFile)
	} else {
		err = cli.client.LoadAliPayPublicKey(cli.aliPublicKey)
	}

	return cli, err
}

func Mode(mode string) ClientOption {
	return func(cli *Client) {
		cli.Mode = mode
	}
}

func Prod(isProd bool) ClientOption {
	return func(cli *Client) {
		cli.IsProd = isProd
	}
}

func PrivateKey(privateKey string) ClientOption {
	return func(cli *Client) {
		cli.privateKey = privateKey
	}
}

func AliPublicKey(aliPublicKey string) ClientOption {
	return func(cli *Client) {
		cli.aliPublicKey = aliPublicKey
	}
}

func CertFiles(appCertSn, rootCertSn, aliPublicCertSn string) ClientOption {
	return func(cli *Client) {
		cli.appCertSnFile = appCertSn
		cli.rootCertSnFile = rootCertSn
		cli.aliPublicCertSnFile = aliPublicCertSn
	}
}

func NotifyURL(uri string) ClientOption {
	return func(cli *Client) {
		cli.NotifyURL = uri
	}
}

func ReturnURL(uri string) ClientOption {
	return func(cli *Client) {
		cli.ReturnURL = uri
	}
}

func WithOrderService(svc unipay.OrderService) ClientOption {
	return func(cli *Client) {
		cli.OrderService = svc
	}
}
