package uniapple

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/awa/go-iap/appstore"
	"github.com/lovewith99/unipay"
)

type ClientOption func(*Client)

func NewClient(password, bundleId string, opts ...ClientOption) *Client {
	cli := &Client{}
	cli.password = password
	cli.bundleID = bundleId

	for _, opt := range opts {
		opt(cli)
	}

	if cli.HttpTimeout == 0 {
		cli.HttpTimeout = 10 * time.Second
	}

	if cli.Locker == nil {
		cli.Locker = unipay.LockerImpl{}
	}

	if cli.AttachService == nil {
		cli.AttachService = unipay.AttachServiceImpl{}
	}

	cli.client = appstore.NewWithClient(&http.Client{
		Timeout: cli.HttpTimeout,
	})
	return cli
}

func HttpTimeout(v time.Duration) ClientOption {
	return func(cli *Client) {
		cli.HttpTimeout = v
	}
}

func WithLocker(locker unipay.Locker) ClientOption {
	return func(cli *Client) {
		cli.Locker = locker
	}
}

func WithAttachService(svc unipay.AttachService) ClientOption {
	return func(cli *Client) {
		cli.AttachService = svc
	}
}

func WithOrderService(svc unipay.IapOrderService) ClientOption {
	return func(cli *Client) {
		cli.OrderService = svc
	}
}

type Client struct {
	Config
	client *appstore.Client

	Locker        unipay.Locker
	OrderService  unipay.IapOrderService
	AttachService unipay.AttachService
}

func (cli *Client) Client() *appstore.Client {
	return cli.client
}

func (cli *Client) VerifyReciept(req *appstore.IAPRequest, retry uint) (*appstore.IAPResponse, error) {
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

func (cli *Client) GetInapp(resp *appstore.IAPResponse, transactionId string) *appstore.InApp {
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

func (cli *Client) CreateInappAttach(transactionId, attach string) error {
	svc := cli.AttachService
	if svc == nil {
		return nil
	}

	if transactionId != "" && attach != "" {
		return svc.Create(transactionId, attach)
	}

	return nil
}

func (cli *Client) DeleteInappAttach(transactionId string) error {
	svc := cli.AttachService
	if svc != nil {
		return svc.Delete(transactionId)
	}

	return nil
}

func (cli *Client) LockInapp(transactionId string) (bool, error) {
	locker := cli.Locker
	if locker != nil {
		return locker.Lock(transactionId)
	}

	return true, nil
}

func (cli *Client) UnLockInapp(transactionId string) error {
	locker := cli.Locker
	if locker != nil {
		return locker.UnLock(transactionId)
	}
	return nil
}

func (cli *Client) Payment(ctx *unipay.Context) error {
	cli.CreateInappAttach(ctx.TransactionId, ctx.Attach)

	ctx.IAPRequest.Password = cli.password
	resp, err := cli.VerifyReciept(&ctx.IAPRequest, 3)
	if err != nil {
		return err
	}

	if resp.Receipt.BundleID != cli.bundleID {
		return errors.New("bundle id mismath")
	}

	inapp := cli.GetInapp(resp, ctx.TransactionId)

	// 小票验证完成，开始处理订单交易
	return cli.Invoke(ctx, inapp)
}

// InvokeHandler 处理小票交易
func (cli *Client) Invoke(ctx *unipay.Context, inapp *appstore.InApp) error {
	if inapp == nil {
		return errors.New("transaction not found")
	}
	ctx.ProductID = inapp.ProductID

	if ok, _ := cli.LockInapp(inapp.TransactionID); !ok {
		// 并发处理同一笔订单, 未获得锁
		return errors.New("concurrency deal: " + inapp.TransactionID)
	}
	defer cli.UnLockInapp(inapp.TransactionID)

	svc := cli.OrderService
	order, err := svc.GetOrderByTradeNo(inapp.TransactionID, unipay.PayWay_AppStore)
	if err != nil {
		err = cli.CheckSubUser(ctx, inapp)
		if err == nil {
			ctx.InApp = inapp
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

	return svc.Invoke(order)
}

func (cli *Client) Revoke(ctx *unipay.Context, inapp *appstore.InApp) error {
	if inapp == nil {
		return errors.New("Transaction not found")
	}
	ctx.ProductID = inapp.ProductID
	if ok, _ := cli.LockInapp(inapp.TransactionID); !ok {
		// 并发处理同一笔订单, 未获得锁
		return errors.New("concurrency deal: " + inapp.TransactionID)
	}
	defer cli.UnLockInapp(inapp.TransactionID)

	svc := cli.OrderService
	order, err := svc.GetOrderByTradeNo(inapp.TransactionID, unipay.PayWay_AppStore)
	if err == nil {
		err = svc.Revoke(order)
	}

	return err
}

func (cli *Client) CheckSubUser(ctx *unipay.Context, inapp *appstore.InApp) error {
	if inapp.OriginalTransactionID == "" {
		return nil
	}

	if inapp.TransactionID == inapp.OriginalTransactionID {
		return nil
	}

	svc := cli.OrderService
	matched := svc.CheckSubUser(ctx, inapp.OriginalTransactionID, inapp.TransactionID)
	if !matched {
		return errors.New("subscribe user mismatch")
	}
	return nil
}

func (cli *Client) AppStoreNotify(ctx *unipay.Context, noti *appstore.SubscriptionNotification, filters ...func(*appstore.InApp) bool) error {
	inapp := GetLatestInapp(noti.UnifiedReceipt.LatestReceiptInfo)

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

func GetLatestInapp(inapps []appstore.InApp) *appstore.InApp {
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
