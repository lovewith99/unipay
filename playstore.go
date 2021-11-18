package unipay

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/awa/go-iap/playstore"
	"google.golang.org/api/androidpublisher/v3"
)

type PlayStoreClientOption func(*PlayStoreClient) error

type PlayStoreClient struct {
	PlayStoreClientConfig

	Locker    OrderLocker
	OrderSvc  IapOrderService
	AttachSvc OrderAttachService
	AndPubSvc AndroidPublisherService
}

func PlayStorePackageName(packageName string) PlayStoreClientOption {
	return func(cli *PlayStoreClient) error {
		cli.PackageName = packageName
		return nil
	}
}

func PlayStorePublicKey(publicKey string) PlayStoreClientOption {
	return func(cli *PlayStoreClient) error {
		cli.publicKey = publicKey
		return nil
	}
}

func PlayStoreOrderLocker(locker OrderLocker) PlayStoreClientOption {
	return func(cli *PlayStoreClient) (err error) {
		cli.Locker = locker
		return
	}
}

func PlayStoreAttachSvc(svc OrderAttachService) PlayStoreClientOption {
	return func(cli *PlayStoreClient) (err error) {
		cli.AttachSvc = svc
		return
	}
}

func PlayStoreOrderSvc(svc IapOrderService) PlayStoreClientOption {
	return func(cli *PlayStoreClient) (err error) {
		cli.OrderSvc = svc
		return
	}
}

func PlayStoreAndroidPublisherSvc(svc AndroidPublisherService) PlayStoreClientOption {
	return func(cli *PlayStoreClient) (err error) {
		if svc == nil {
			err = errors.New("AndroidPublisherService is nil")
		} else {
			cli.AndPubSvc = svc
		}
		return
	}
}

func (cli *PlayStoreClient) VerifyPurchaseDataSign(purchaseData []byte, sign string) error {
	ok, err := playstore.VerifySignature(cli.publicKey, purchaseData, sign)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("invalid purchase data sign")
	}

	return nil
}

func (cli *PlayStoreClient) IapPayment(ctx *Context) error {
	// step1: 验证签名
	purchaseData := []byte(ctx.PurchaseData)
	err := cli.VerifyPurchaseDataSign(purchaseData, ctx.PurchaseDataSign)
	if err != nil {
		return err
	}

	// todo: 向google play store服务器发送订单状态查询, 确认订单已支付
	var inapp InAppPurchaseData
	err = json.Unmarshal(purchaseData, &inapp)
	if err != nil {
		return err
	}

	if inapp.PackageName != cli.PackageName {
		return errors.New("package name mismatch")
	}

	return cli.Invoke(ctx, &inapp)
}

func (cli *PlayStoreClient) SetOriOrderId(inapp *InAppPurchaseData) error {
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

func (cli *PlayStoreClient) SetSubscriptionPurchase(inapp *InAppPurchaseData) error {
	if inapp.SubscriptionPurchase != nil {
		return nil
	}

	svc := cli.AndPubSvc
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

func (cli *PlayStoreClient) Revoke(ctx *Context, inapp *InAppPurchaseData) error {
	if inapp == nil {
		return errors.New("Transaction not found")
	}
	ctx.ProductID = inapp.ProductId
	cli.SetOriOrderId(inapp)

	if ok, _ := cli.LockTransaction(inapp.OrderId); !ok {
		return errors.New("concurrenty deal: " + inapp.OrderId)
	}
	defer cli.UnLockTransaction(inapp.OrderId)

	svc := cli.OrderSvc
	order, err := svc.GetOrderByTradeNo(inapp.OrderId, PayWay_PlayStore)
	if err == nil {
		err = svc.Revoke(order)
	}

	return err
}

func (cli *PlayStoreClient) Invoke(ctx *Context, inapp *InAppPurchaseData) error {
	if inapp == nil {
		return errors.New("Transaction not found")
	}
	ctx.ProductID = inapp.ProductId
	cli.SetOriOrderId(inapp)

	if ok, _ := cli.LockTransaction(inapp.OrderId); !ok {
		return errors.New("concurrenty deal: " + inapp.OrderId)
	}
	defer cli.UnLockTransaction(inapp.OrderId)

	svc := cli.OrderSvc
	order, err := svc.GetOrderByTradeNo(inapp.OrderId, PayWay_PlayStore)
	if err != nil {
		err = cli.CheckSubUser(ctx, inapp)
		if err == nil {
			ctx.SetInApp(inapp)
			order, err = svc.PostOrder(ctx)
		}
	}

	if err != nil {
		return err
	}

	// 订单已处理，直接返回
	if order.Payed() {
		return nil
	}

	return svc.PayCB(order)
}

func (cli *PlayStoreClient) CheckSubUser(ctx *Context, inapp *InAppPurchaseData) error {
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

	svc := cli.OrderSvc
	matched := svc.CheckSubUser(ctx, inapp.OriOrderId, inapp.OrderId)
	if !matched {
		return errors.New("subscribe user mismatch")
	}
	return nil
}

func (cli *PlayStoreClient) LockTransaction(transactionId string) (bool, error) {
	locker := cli.Locker
	if locker != nil {
		return locker.Lock(transactionId)
	}

	return true, nil
}

func (cli *PlayStoreClient) UnLockTransaction(transactionId string) error {
	locker := cli.Locker
	if locker != nil {
		return locker.UnLock(transactionId)
	}
	return nil
}

func (cli *PlayStoreClient) PlayStoreNotify(ctx *Context, noti *RTDNotification, filters ...func(*DeveloperNotification) bool) error {
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

func (cli *PlayStoreClient) OneTimeProductNotify(ctx *Context, noti *OneTimeProductNotification) error {
	// switch noti.NotificationType {
	// case ONE_TIME_PRODUCT_PURCHASED:
	// case ONE_TIME_PRODUCT_CANCELED:
	// }
	if noti.NotificationType == ONE_TIME_PRODUCT_CANCELED {
		return nil
	}

	svc := cli.AndPubSvc
	data, err := svc.VerifyProduct(context.Background(),
		cli.PackageName, noti.Sku, noti.PurchaseToken)
	if err != nil {
		return err
	}

	if data.PurchaseState == 0 && data.AcknowledgementState == 0 {
		// 已购买
		purchaseData := InAppPurchaseData{
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

func (cli *PlayStoreClient) SubscriptionNotify(ctx *Context, noti *SubscriptionNotification) error {
	svc := cli.AndPubSvc
	data, err := svc.VerifySubscription(
		context.Background(),
		cli.PackageName,
		noti.SubscriptionId,
		noti.PurchaseToken,
	)
	if err != nil {
		return err
	}

	purchaseData := InAppPurchaseData{
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

// doc: https://developer.android.com/google/play/billing/billing_reference
type InAppPurchaseData struct {
	// 表明是否自动续订订阅。如果为 true，则表示订阅处于活动状态，并将在下一个结算日期自动续订。
	// 如果为 false，则表示用户已取消订阅。用户可以在下一个结算日期之前访问订阅内容，并且在该日期后将无法访问，
	// 除非他们重新启用自动续订（或者手动续订，如手动续订中所述）。如果您提供宽限期，只要宽限期尚未结束，
	// 对于所有订阅而言，此值都将保持为 true。下一次结算日期每天都会自动推延，直至宽限期结束或用户更改他们的付款方式。
	AutoRenewing bool `json:"autoRenewing"`

	OrderId          string `json:"orderId"`          // 交易的唯一订单标识符。
	PackageName      string `json:"packageName"`      // 发起购买的应用软件包。
	ProductId        string `json:"productId"`        // 商品的产品标识符。
	PurchaseTime     int64  `json:"purchaseTime"`     // 购买产品的时间，单位毫秒。
	PurchaseState    int    `json:"purchaseState"`    // 订单的购买状态。始终返回 0（已购买）。
	DeveloperPayload string `json:"developerPayload"` // 开发者指定的字符串，其中包含关于订单的补充信息。
	PurchaseToken    string `json:"purchaseToken"`    // 用于对给定商品和用户对的购买交易进行唯一标识的令牌

	OriOrderId           string                                 `json:"-"` // 连续订阅的第一笔订阅id
	SubscriptionPurchase *androidpublisher.SubscriptionPurchase `json:"-"`
}

func NewPlayStoreClient(opts ...PlayStoreClientOption) (*PlayStoreClient, error) {
	var err error
	client := &PlayStoreClient{}
	for _, opt := range opts {
		err = opt(client)
		if err != nil {
			break
		}
	}

	return client, err
}
