package iap

import "github.com/awa/go-iap/appstore"

func GetTradeNo(inapp interface{}) (no string, orino string) {
	switch inapp.(type) {
	case *appstore.InApp:
		v := inapp.(*appstore.InApp)
		no = v.TransactionID
		orino = v.OriginalTransactionID
	case *PurchaseData:
		v := inapp.(*PurchaseData)
		no = v.OrderId
		orino = v.OriOrderId
	}
	return
}

// IsFirstSub 是否是首次订阅
func IsFirstSub(inapp interface{}) bool {
	no, orino := GetTradeNo(inapp)

	if no == "" || orino == "" {
		return false
	}

	return no == orino
}

// IsFreeTrial 是否是免费试用
func IsFreeTrial(inapp interface{}) bool {
	switch inapp.(type) {
	case *appstore.InApp:
		v := inapp.(*appstore.InApp)
		return v.IsTrialPeriod == "true"
	case *PurchaseData:
		v := inapp.(*PurchaseData)
		if v.SubscriptionPurchase != nil {
			return v.SubscriptionPurchase.PaymentState == 2
		}

		// todo: 这里的判断并不准确
		// 准确的办法是在PayOrderContext设置Inapp之前
		// 拿PurchaseToken和productId到play store获取
		// SubscriptionPurchase 详细数据
		return v.OrderId == v.OriOrderId
	}
	return false
}
