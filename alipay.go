package unipay

import (
	"fmt"

	alipayv3 "github.com/smartwalle/alipay/v3"
)

type AliPayClient struct {
	AliPayClientConfig

	client    *alipayv3.Client
	OrderSvc  UniPayOrderService
	OrderInfo func(UniPayOrder) *UniPayOrderInfo
}

func (cli *AliPayClient) Client() *alipayv3.Client {
	return cli.client
}

func (cli *AliPayClient) AppPayment(ctx *Context) (MapResult, error) {
	svc := cli.OrderSvc

	// order, err := svc.PostOrder(ctx)
	order, err := svc.PostOrder(ctx)
	if err != nil {
		return nil, err
	}

	obj := alipayv3.TradeAppPay{}
	obj.ProductCode = "QUICK_MSECURITY_PAY"
	obj.NotifyURL = cli.NotifyURL

	info := cli.OrderInfo(order)
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

func (cli *AliPayClient) WapPayment(ctx *Context) (MapResult, error) {
	svc := cli.OrderSvc

	order, err := svc.PostOrder(ctx)
	if err != nil {
		return nil, err
	}

	obj := alipayv3.TradeWapPay{}
	obj.ProductCode = "QUICK_WAP_WAY"
	obj.ReturnURL = cli.ReturnURL
	obj.NotifyURL = cli.NotifyURL

	info := cli.OrderInfo(order)
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

type AliPayClientOption func(*AliPayClient)

func NewAliPayClient(opts ...AliPayClientOption) (*AliPayClient, error) {
	var err error
	cli := &AliPayClient{}

	for _, opt := range opts {
		opt(cli)
	}

	cli.client, err = alipayv3.New(cli.appId, cli.privateKey, cli.IsProd)
	if err != nil {
		return nil, err
	}

	if cli.Mode == AliPay_CertMode {
		cli.client.LoadAppPublicCertFromFile(cli.appCertSnFile)
		cli.client.LoadAliPayRootCertFromFile(cli.rootCertSnFile)
		cli.client.LoadAliPayPublicCertFromFile(cli.aliPublicCertSnFile)
	} else {
		err = cli.client.LoadAliPayPublicKey(cli.aliPublicKey)
	}

	return cli, err
}

func AliPayConfig(appId, partnerId string) AliPayClientOption {
	return func(cli *AliPayClient) {
		cli.appId = appId
		cli.partnerId = partnerId
	}
}

func AliPayPrivateKey(privateKey string) AliPayClientOption {
	return func(cli *AliPayClient) {
		cli.privateKey = privateKey
	}
}

func AliPayAliPublicKey(aliPublicKey string) AliPayClientOption {
	return func(cli *AliPayClient) {
		cli.aliPublicKey = aliPublicKey
	}
}

func AliPayCertFile(appCertSn, rootCertSn, aliPublicCertSn string) AliPayClientOption {
	return func(cli *AliPayClient) {
		cli.appCertSnFile = appCertSn
		cli.rootCertSnFile = rootCertSn
		cli.aliPublicCertSnFile = aliPublicCertSn
	}
}

func AliPayMode(mode string) AliPayClientOption {
	return func(cli *AliPayClient) {
		cli.Mode = mode
	}
}

func AliPayEnv(isProd bool) AliPayClientOption {
	return func(cli *AliPayClient) {
		cli.IsProd = isProd
	}
}

func AliPayOrderSvc(svc UniPayOrderService) AliPayClientOption {
	return func(cli *AliPayClient) {
		cli.OrderSvc = svc
	}
}

func AliPayNotifyURL(uri string) AliPayClientOption {
	return func(cli *AliPayClient) {
		cli.NotifyURL = uri
	}
}

func AliPayReturnURL(uri string) AliPayClientOption {
	return func(cli *AliPayClient) {
		cli.ReturnURL = uri
	}
}

func AliPayGetOrderInfoFunc(f func(UniPayOrder) *UniPayOrderInfo) AliPayClientOption {
	return func(cli *AliPayClient) {
		cli.OrderInfo = f
	}
}
