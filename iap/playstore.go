package iap

import "google.golang.org/api/androidpublisher/v3"

// doc: https://developer.android.com/google/play/billing/billing_reference
type PurchaseData struct {
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
