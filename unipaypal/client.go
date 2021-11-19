package unipaypal

import (
	"context"
	"fmt"

	"github.com/lovewith99/unipay"
	paypal "github.com/plutov/paypal/v4"
)

type Client struct {
	Config
	client *paypal.Client

	OrderService unipay.OrderService
}

type ClientOption func(*Client)

func NotifyURL(returnURL, cancelURL string) ClientOption {
	return func(cli *Client) {
		cli.ReturnURL = returnURL
		cli.CancelURL = cancelURL
	}
}

func WithOrderService(svc unipay.OrderService) ClientOption {
	return func(cli *Client) {
		cli.OrderService = svc
	}
}

func NewClient(prod bool, clientId, secret string, opts ...ClientOption) (*Client, error) {
	var err error
	client := &Client{}
	client.IsProd = prod
	client.clientId = clientId
	client.secret = secret

	for _, opt := range opts {
		opt(client)
	}

	apiBase := paypal.APIBaseSandBox
	if client.IsProd {
		apiBase = paypal.APIBaseLive
	}

	client.client, err = paypal.NewClient(clientId, secret, apiBase)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (cli *Client) Client() *paypal.Client {
	return cli.client
}

func (cli *Client) GetAccessToken() (*paypal.TokenResponse, error) {
	c := cli.client

	if c.Token != nil {
		return c.Token, nil
	}

	c.Lock()
	defer c.Unlock()

	return c.GetAccessToken(context.Background())
}

func (cli *Client) CreateOrder(ctx *unipay.Context, order unipay.IOrder) (*paypal.Order, error) {
	// info := cli.OrderInfo(order)
	info := order.OrderInfo()
	amount := fmt.Sprintf("%.2f", float64(info.TotalFee)/100)

	c := cli.client
	purchaseUnits := []paypal.PurchaseUnitRequest{
		{
			InvoiceID: info.OutTradeNo,
			CustomID:  info.Attach,
			Amount: &paypal.PurchaseUnitAmount{
				Currency: info.Currency,
				Value:    amount,
				Breakdown: &paypal.PurchaseUnitAmountBreakdown{
					ItemTotal: &paypal.Money{
						Currency: info.Currency,
						Value:    amount,
					},
				},
			},
			Items: []paypal.Item{
				{
					Name:     info.Subject,
					Quantity: "1",
					UnitAmount: &paypal.Money{
						Currency: info.Currency,
						Value:    amount,
					},
				},
			},
		},
	}
	payer := &paypal.CreateOrderPayer{}
	appCtx := &paypal.ApplicationContext{
		ReturnURL: cli.ReturnURL,
		CancelURL: cli.CancelURL,
	}

	return c.CreateOrder(context.Background(), "CAPTURE", purchaseUnits, payer, appCtx)
}

func (cli *Client) Payment(ctx *unipay.Context) (unipay.MapResult, error) {
	// paypal.PaymentPayer
	svc := cli.OrderService

	order, err := svc.PostOrder(ctx)
	if err != nil {
		return nil, err
	}

	_, err = cli.GetAccessToken()
	if err != nil {
		return nil, err
	}

	// 创建paypel订单
	pporder, err := cli.CreateOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	if pporder.Status != "CREATED" {
		return nil, fmt.Errorf("paypal checkout order status: %s", pporder.Status)
	}

	result := unipay.MapResult{
		"id":          pporder.ID,
		"status":      pporder.Status,
		"links":       pporder.Links,
		"approve_url": "",
	}

	for _, e := range pporder.Links {
		if e.Rel == "approve" {
			result["approve_url"] = e.Href
			break
		}
	}

	return result, nil
}

func (cli *Client) CapturePaymentOrder(orderId string) (*paypal.CaptureOrderResponse, error) {
	_, err := cli.GetAccessToken()
	if err != nil {
		return nil, err
	}

	c := cli.client
	capture := paypal.CaptureOrderRequest{}
	resp, err := c.CaptureOrder(context.Background(), orderId, capture)
	if err != nil {
		return nil, err
	}

	// 1. CREATED. The order was created with the specified context.
	// 1. SAVED. The order was saved and persisted. The order status continues to be in progress until a capture is made with final_capture = true for all purchase units within the order.
	// 3. APPROVED. The customer approved the payment through the PayPal wallet or another form of guest or unbranded payment. For example, a card, bank account, or so on.
	// 4. VOIDED. All purchase units in the order are voided.
	// 5. COMPLETED. The payment was authorized or the authorized payment was captured for the order.
	// 6. PAYER_ACTION_REQUIRED. The order requires an action from the payer (e.g. 3DS authentication). Redirect the payer to the "rel":"payer-action" HATEOAS link returned as part of the response prior to authorizing or capturing the order.
	// switch resp.Status {
	// case "CREATED":
	// case "SAVED":
	// case "APPROVED":
	// case "VOIDED":
	// case "COMPLETED":
	// case "PAYER_ACTION_REQUIRED":
	// }

	return resp, nil
}
