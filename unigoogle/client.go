package unigoogle

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/awa/go-iap/playstore"
	"github.com/lovewith99/unipay"
	"github.com/lovewith99/unipay/iap"
	"google.golang.org/api/androidpublisher/v3"
)

type ClientOption func(*Client) error

type Client struct {
	Config

	Locker          unipay.Locker
	OrderService    unipay.IapOrderService
	AttachService   unipay.AttachService
	PubliserService PublisherService
}

func NewClient(opts ...ClientOption) (*Client, error) {
	var err error
	client := &Client{}
	for _, opt := range opts {
		err = opt(client)
		if err != nil {
			break
		}
	}

	if client.Locker == nil {
		client.Locker = unipay.LockerImpl{}
	}

	if client.AttachService == nil {
		client.AttachService = unipay.AttachServiceImpl{}
	}

	return client, err
}

func PackageName(packageName string) ClientOption {
	return func(cli *Client) error {
		cli.PackageName = packageName
		return nil
	}
}

func PublicKey(publicKey string) ClientOption {
	return func(cli *Client) error {
		cli.publicKey = publicKey
		return nil
	}
}

func WithLocker(locker unipay.Locker) ClientOption {
	return func(cli *Client) (err error) {
		cli.Locker = locker
		return
	}
}

func WithAttachService(svc unipay.AttachService) ClientOption {
	return func(cli *Client) (err error) {
		cli.AttachService = svc
		return
	}
}

func WithOrderService(svc unipay.IapOrderService) ClientOption {
	return func(cli *Client) (err error) {
		cli.OrderService = svc
		return
	}
}

func WithPublisherService(svc PublisherService) ClientOption {
	return func(cli *Client) (err error) {
		if svc == nil {
			err = errors.New("PublisherService is nil")
		} else {
			cli.PubliserService = svc
		}
		return
	}
}

func (cli *Client) VerifyPurchaseDataSign(purchaseData []byte, sign string) error {
	ok, err := playstore.VerifySignature(cli.publicKey, purchaseData, sign)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("invalid purchase data sign")
	}

	return nil
}

func (cli *Client) Payment(ctx *unipay.Context) error {
	// step1: 验证签名
	purchaseData := []byte(ctx.PurchaseData)
	err := cli.VerifyPurchaseDataSign(purchaseData, ctx.PurchaseDataSign)
	if err != nil {
		return err
	}

	// todo: 向google play store服务器发送订单状态查询, 确认订单已支付
	var inapp iap.PurchaseData
	err = json.Unmarshal(purchaseData, &inapp)
	if err != nil {
		return err
	}

	if inapp.PackageName != cli.PackageName {
		return errors.New("package name mismatch")
	}

	return cli.Invoke(ctx, &inapp)
}

func (cli *Client) SetOriOrderId(inapp *iap.PurchaseData) error {
	if inapp.OriOrderId != "" {
		return nil
	}

	specs := strings.Split(inapp.OrderId, "..")
	if len(specs) == 1 {
		inapp.OriOrderId = inapp.OrderId
	}

	if len(specs) == 2 {
		inapp.OriOrderId = specs[0]
	}

	return nil
}

func (cli *Client) SetSubscriptionPurchase(inapp *iap.PurchaseData) error {
	if inapp.SubscriptionPurchase != nil {
		return nil
	}

	svc := cli.PubliserService
	data, err := svc.VerifySubscription(
		context.Background(),
		cli.PackageName,
		inapp.ProductId,
		inapp.PurchaseToken,
	)
	if err != nil {
		return err
	}

	inapp.SubscriptionPurchase = data
	return nil
}

func (cli *Client) Revoke(ctx *unipay.Context, inapp *iap.PurchaseData) error {
	if inapp == nil {
		// return errors.New("Transaction not found")
		return unipay.OrderNotFoundError
	}
	ctx.ProductID = inapp.ProductId
	cli.SetOriOrderId(inapp)

	if ok, _ := cli.LockOrder(inapp.OrderId); !ok {
		return errors.New("concurrency deal: " + inapp.OrderId)
	}
	defer cli.UnLockOrder(inapp.OrderId)

	svc := cli.OrderService
	order, err := svc.GetOrderByTradeNo(inapp.OrderId, unipay.PayWay_PlayStore)
	if err == nil {
		err = svc.Revoke(order)
	}

	return err
}

func (cli *Client) Invoke(ctx *unipay.Context, inapp *iap.PurchaseData) error {
	if inapp == nil {
		// return errors.New("Transaction not found")
		return unipay.OrderNotFoundError
	}
	ctx.ProductID = inapp.ProductId
	cli.SetOriOrderId(inapp)

	if ok, _ := cli.LockOrder(inapp.OrderId); !ok {
		return errors.New("concurrency deal: " + inapp.OrderId)
	}
	defer cli.UnLockOrder(inapp.OrderId)

	svc := cli.OrderService
	order, err := svc.GetOrderByTradeNo(inapp.OrderId, unipay.PayWay_PlayStore)
	if err != nil {
		if err := cli.CheckSubUser(ctx, inapp); err != nil {
			return err
		}
		ctx.InApp = inapp
		order, err = svc.PostOrder(ctx)
		if err != nil {
			return err
		}
	}

	// if err != nil {
	// 	return err
	// }
	// 订单已处理，直接返回
	if order.Payed() {
		return nil
	}

	return svc.Invoke(order)
}

func (cli *Client) CheckSubUser(ctx *unipay.Context, inapp *iap.PurchaseData) error {
	// if !inapp.AutoRenewing {
	// 	return nil
	// }

	if inapp.OrderId == inapp.OriOrderId {
		return nil
	}

	// https://developer.android.com/google/play/billing/integrate
	// 判断是不是续订
	// 订阅续订的订单号包含一个额外的整数，它表示具体是第几次续订。
	// 例如，初始订阅的订单 ID 可能是 GPA.1234-5678-9012-34567，
	// 后续订单 ID 是 GPA.1234-5678-9012-34567..0（第一次续订）、
	// GPA.1234-5678-9012-34567..1（第二次续订），依此类推。

	return cli.OrderService.CheckSubUser(ctx, inapp.OriOrderId, inapp.OrderId)
}

func (cli *Client) LockOrder(transactionId string) (bool, error) {
	locker := cli.Locker
	if locker != nil {
		return locker.Lock(transactionId)
	}

	return true, nil
}

func (cli *Client) UnLockOrder(transactionId string) error {
	locker := cli.Locker
	if locker != nil {
		return locker.UnLock(transactionId)
	}
	return nil
}

func (cli *Client) PlayStoreNotify(ctx *unipay.Context, noti *RTDNotification, filters ...func(*DeveloperNotification) bool) error {
	dn, err := noti.GetDeveloperNotification()
	if err != nil {
		return err
	}

	// 过滤通知
	for _, filter := range filters {
		if ok := filter(dn); !ok {
			return nil
		}
	}

	if dn.OneTimeProductNotification.NotificationType > 0 {
		return cli.OneTimeProductNotify(ctx, &dn.OneTimeProductNotification)
	}

	if dn.SubscriptionNotification.NotificationType > 0 {
		return cli.SubscriptionNotify(ctx, &dn.SubscriptionNotification)
	}

	return nil
}

func (cli *Client) OneTimeProductNotify(ctx *unipay.Context, noti *OneTimeProductNotification) error {
	// switch noti.NotificationType {
	// case ONE_TIME_PRODUCT_PURCHASED:
	// case ONE_TIME_PRODUCT_CANCELED:
	// }
	if noti.NotificationType == ONE_TIME_PRODUCT_CANCELED {
		return nil
	}

	svc := cli.PubliserService
	data, err := svc.VerifyProduct(context.Background(),
		cli.PackageName, noti.Sku, noti.PurchaseToken)
	if err != nil {
		return err
	}

	if data.PurchaseState == 0 && data.AcknowledgementState == 0 {
		// 已购买
		purchaseData := iap.PurchaseData{
			AutoRenewing:     false,
			PackageName:      cli.PackageName,
			OrderId:          data.OrderId,
			ProductId:        data.ProductId,
			PurchaseToken:    data.PurchaseToken,
			DeveloperPayload: data.DeveloperPayload,
			PurchaseState:    int(data.PurchaseState),
		}
		cli.SetOriOrderId(&purchaseData)
		err = cli.Invoke(ctx, &purchaseData)
		if err == nil {
			err = svc.AcknowledgeProduct(context.Background(),
				cli.PackageName, data.ProductId, data.PurchaseToken,
				data.DeveloperPayload,
			)
		}
	}

	return err
}

func (cli *Client) SubscriptionNotify(ctx *unipay.Context, noti *SubscriptionNotification) error {
	svc := cli.PubliserService
	data, err := svc.VerifySubscription(
		context.Background(),
		cli.PackageName,
		noti.SubscriptionId,
		noti.PurchaseToken,
	)
	if err != nil {
		return err
	}

	purchaseData := iap.PurchaseData{
		AutoRenewing:         data.AutoRenewing,
		PackageName:          cli.PackageName,
		OrderId:              data.OrderId,
		ProductId:            noti.SubscriptionId,
		PurchaseToken:        noti.PurchaseToken,
		DeveloperPayload:     data.DeveloperPayload,
		SubscriptionPurchase: data,
	}
	cli.SetOriOrderId(&purchaseData)

	switch noti.NotificationType {
	case SUBSCRIPTION_RECOVERED:
		if data.PaymentState == 1 {
			err = cli.Invoke(ctx, &purchaseData)
		}
	case SUBSCRIPTION_RENEWED:
		if data.PaymentState == 1 {
			err = cli.Invoke(ctx, &purchaseData)
		}
	case SUBSCRIPTION_RESTARTED:
		if data.PaymentState == 1 {
			err = cli.Invoke(ctx, &purchaseData)
		}
	case SUBSCRIPTION_PURCHASED:
		if data.PaymentState == 1 || data.PaymentState == 2 {
			err = cli.Invoke(ctx, &purchaseData)
		}
	case SUBSCRIPTION_REVOKED:
		if data.PaymentState == 1 || data.PaymentState == 2 {
			err = cli.Revoke(ctx, &purchaseData)
		}
	}

	if data.AcknowledgementState == 0 && err == nil {
		err = svc.AcknowledgeSubscription(
			context.Background(),
			cli.PackageName,
			noti.SubscriptionId,
			noti.PurchaseToken,
			&androidpublisher.SubscriptionPurchasesAcknowledgeRequest{
				DeveloperPayload: data.DeveloperPayload,
				ForceSendFields:  data.ForceSendFields,
				NullFields:       data.NullFields,
			},
		)
	}

	return err
}
