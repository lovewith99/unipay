package unipay

import (
	"github.com/awa/go-iap/appstore"
)

type MapResult map[string]interface{}

type Context struct {
	Request

	InApp    InAppData   `json:"-"`
	Uid      interface{} `json:"-"`
	ClientIP string      `json:"-"`
	Currency string      `json:"-"`
}

func PayContext(payWay uint8) *Context {
	ctx := &Context{}
	ctx.PayWay = payWay
	return ctx
}

const (
	_ = iota
	InApp_AppStore
	InApp_PlayStore
)

type InAppData struct {
	kind int

	AppStoreInapp  *appstore.InApp
	PlayStoreInapp *InAppPurchaseData
}

func (ctx *Context) SetInApp(inapp interface{}) {
	switch inapp.(type) {
	case *appstore.InApp:
		ctx.InApp.kind = InApp_AppStore
		ctx.InApp.AppStoreInapp = inapp.(*appstore.InApp)
	case *InAppPurchaseData:
		ctx.InApp.kind = InApp_PlayStore
		ctx.InApp.PlayStoreInapp = inapp.(*InAppPurchaseData)
	}
}

func (ctx *Context) GetTradeNo() (no string, orino string) {
	switch ctx.InApp.kind {
	case InApp_AppStore:
		inapp := ctx.InApp.AppStoreInapp
		no = inapp.TransactionID
		orino = inapp.OriginalTransactionID
	case InApp_PlayStore:
		inapp := ctx.InApp.PlayStoreInapp
		no = inapp.OrderId
		orino = inapp.OriOrderId
	}

	return
}

// IsFirstSub 是否是首次订阅
func (ctx *Context) IsFirstSub() bool {
	switch ctx.InApp.kind {
	case InApp_AppStore:
		inapp := ctx.InApp.AppStoreInapp
		return inapp.TransactionID == inapp.OriginalTransactionID
	case InApp_PlayStore:
		inapp := ctx.InApp.PlayStoreInapp
		return inapp.OrderId == inapp.OriOrderId
	}

	return false
}

// IsFreeTrial 是否是免费试用
func (ctx *Context) IsFreeTrial() bool {
	switch ctx.InApp.kind {
	case InApp_AppStore:
		inapp := ctx.InApp.AppStoreInapp
		return inapp.IsTrialPeriod == "true"
	case InApp_PlayStore:
		inapp := ctx.InApp.PlayStoreInapp
		if inapp.SubscriptionPurchase != nil {
			return inapp.SubscriptionPurchase.PaymentState == 2
		}

		// todo: 这里的判断并不准确
		// 准确的办法是在PayOrderContext设置Inapp之前
		// 拿PurchaseToken和productId到play store获取
		// SubscriptionPurchase 详细数据
		return inapp.OrderId == inapp.OriOrderId
	}

	return false
}

func (ctx *Context) IsTrialPeriod() bool {
	switch ctx.InApp.kind {
	case InApp_AppStore:
		inapp := ctx.InApp.AppStoreInapp
		return inapp.IsTrialPeriod == "true"
	case InApp_PlayStore:
		inapp := ctx.InApp.PlayStoreInapp
		if sub := inapp.SubscriptionPurchase; sub != nil {
			return sub.PaymentState == 2
		}
	}

	return false
}
