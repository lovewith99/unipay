package unipay

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/awa/go-iap/appstore"
)

type AppStoreClientOption func(*AppStoreClient)

func NewAppStoreClient(password, bundleId string, opts ...AppStoreClientOption) *AppStoreClient {
	cli := &AppStoreClient{}
	cli.password = password
	cli.bundleID = bundleId

	for _, opt := range opts {
		opt(cli)
	}

	if cli.HttpTimeout == 0 {
		cli.HttpTimeout = 10 * time.Second
	}

	if cli.Locker == nil {
		cli.Locker = EmptyOrderLocker{}
	}

	if cli.AttachSvc == nil {
		cli.AttachSvc = EmptyOrderAttachService{}
	}

	cli.client = appstore.NewWithClient(&http.Client{
		Timeout: cli.HttpTimeout,
	})
	return cli
}

func AppStoreOrderLocker(locker OrderLocker) AppStoreClientOption {
	return func(cli *AppStoreClient) {
		cli.Locker = locker
	}
}

func AppStoreAttachSvc(svc OrderAttachService) AppStoreClientOption {
	return func(cli *AppStoreClient) {
		cli.AttachSvc = svc
	}
}

func AppStoreOrderSvc(svc IapOrderService) AppStoreClientOption {
	return func(cli *AppStoreClient) {
		cli.OrderSvc = svc
	}
}

type AppStoreClient struct {
	AppStoreClientConfig
	client *appstore.Client

	Locker OrderLocker

	// 用来保存苹果订单的附件信息，
	// 避免补单时信息丢失
	AttachSvc OrderAttachService

	OrderSvc IapOrderService
}

func (cli *AppStoreClient) Client() *appstore.Client {
	return cli.client
}

func (cli *AppStoreClient) VerifyReciept(req *appstore.IAPRequest, retry uint) (*appstore.IAPResponse, error) {
	var err error
	resp := &appstore.IAPResponse{}

	for retry >= 0 {
		retry--
		err = cli.client.Verify(context.Background(), *req, resp)
		if err == nil {
			break
		}
	}

	if err != nil {
		return resp, err
	}

	err = appstore.HandleError(resp.Status)
	return resp, err
}

func (cli *AppStoreClient) GetTransaction(resp *appstore.IAPResponse, transactionId string) *appstore.InApp {
	var transaction *appstore.InApp
	for i := range resp.LatestReceiptInfo {
		if resp.LatestReceiptInfo[i].TransactionID == transactionId {
			transaction = &resp.LatestReceiptInfo[i]
			break
		}
	}

	if transaction == nil {
		for i := range resp.Receipt.InApp {
			if resp.Receipt.InApp[i].TransactionID == transactionId {
				transaction = &resp.Receipt.InApp[i]
				break
			}
		}
	}

	return transaction
}

func (cli *AppStoreClient) CreateTrasactionAttach(transactionId, attach string) error {
	svc := cli.AttachSvc
	if svc == nil {
		return nil
	}

	if transactionId != "" && attach != "" {
		return svc.Create(transactionId, attach)
	}

	return nil
}

func (cli *AppStoreClient) DeleteTrasactionAttach(transactionId string) error {
	svc := cli.AttachSvc
	if svc != nil {
		return svc.Delete(transactionId)
	}

	return nil
}

func (cli *AppStoreClient) LockTransaction(transactionId string) (bool, error) {
	locker := cli.Locker
	if locker != nil {
		return locker.Lock(transactionId)
	}

	return true, nil
}

func (cli *AppStoreClient) UnLockTransaction(transactionId string) error {
	locker := cli.Locker
	if locker != nil {
		return locker.UnLock(transactionId)
	}
	return nil
}

func (cli *AppStoreClient) IapPayment(ctx *Context) error {
	cli.CreateTrasactionAttach(ctx.TransactionId, ctx.Attach)

	ctx.IAPRequest.Password = cli.password
	resp, err := cli.VerifyReciept(&ctx.IAPRequest, 3)
	if err != nil {
		return err
	}

	if resp.Receipt.BundleID != cli.bundleID {
		return errors.New("bundle id mismath")
	}

	inapp := cli.GetTransaction(resp, ctx.TransactionId)

	// 小票验证完成，开始处理订单交易
	return cli.Invoke(ctx, inapp)
}

func (cli *AppStoreClient) Revoke(ctx *Context, inapp *appstore.InApp) error {
	if inapp == nil {
		return errors.New("Transaction not found")
	}
	ctx.ProductID = inapp.ProductID
	if ok, _ := cli.LockTransaction(inapp.TransactionID); !ok {
		// 并发处理同一笔订单, 未获得锁
		return errors.New("concurrenty deal: " + inapp.TransactionID)
	}
	defer cli.UnLockTransaction(inapp.TransactionID)

	svc := cli.OrderSvc
	order, err := svc.GetOrderByTradeNo(inapp.TransactionID, PayWay_AppStore)
	if err == nil {
		err = svc.Revoke(order)
	}

	return err
}

// InvokeHandler 处理小票交易
func (cli *AppStoreClient) Invoke(ctx *Context, inapp *appstore.InApp) error {
	if inapp == nil {
		return errors.New("transaction not found")
	}
	ctx.ProductID = inapp.ProductID

	if ok, _ := cli.LockTransaction(inapp.TransactionID); !ok {
		// 并发处理同一笔订单, 未获得锁
		return errors.New("concurrenty deal: " + inapp.TransactionID)
	}
	defer cli.UnLockTransaction(inapp.TransactionID)

	svc := cli.OrderSvc
	order, err := svc.GetOrderByTradeNo(inapp.TransactionID, PayWay_AppStore)
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

func (cli *AppStoreClient) CheckSubUser(ctx *Context, inapp *appstore.InApp) error {
	if inapp.OriginalTransactionID == "" {
		return nil
	}

	if inapp.TransactionID == inapp.OriginalTransactionID {
		return nil
	}

	svc := cli.OrderSvc
	matched := svc.CheckSubUser(ctx, inapp.OriginalTransactionID, inapp.TransactionID)
	if !matched {
		return errors.New("subscribe user mismatch")
	}
	return nil
}

func (cli *AppStoreClient) AppStoreNotify(ctx *Context, noti *appstore.SubscriptionNotification, filters ...func(*appstore.InApp) bool) error {
	inapp := GetLatestTranscation(noti.UnifiedReceipt.LatestReceiptInfo)

	for _, filter := range filters {
		if ok := filter(inapp); !ok {
			return nil
		}
	}

	switch noti.NotificationType {
	case appstore.NotificationTypeDidRecover:
		// 过期的订阅成功恢复订阅之后的通知
		return cli.Invoke(ctx, inapp)
	case appstore.NotificationTypeDidRenew:
		// 订阅期内自动订阅成功通知
		return cli.Invoke(ctx, inapp)
	case appstore.NotificationTypeInitialBuy:
		// 首次订阅通知, 不处理, 由客户端调用处理
		// ctx.InApp = svc.GetLatestTranscation(noti.UnifiedReceipt.LatestReceiptInfo)
	case appstore.NotificationTypeInteractiveRenewal:
		// data = GetLatestTranscation(noti.UnifiedReceipt.LatestReceiptInfo)
		return cli.Invoke(ctx, inapp)
	case appstore.NotificationTypeRefund:

	case appstore.NotificationTypeRenewal: // 2021.03.10之后appstore不再发送此类型的通知
	}

	return nil
}

func GetLatestTranscation(inapps []appstore.InApp) *appstore.InApp {
	var ts int64
	var inapp *appstore.InApp
	for i := range inapps {
		e := &inapps[i]
		pms, _ := strconv.ParseInt(e.PurchaseDateMS, 10, 64)
		if pms > ts {
			ts = pms
			inapp = e
		}
	}

	return inapp
}
