package playunipay

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/awa/go-iap/playstore"
)

// google订阅的通知类型
const (
	SUBSCRIPTION_RECOVERED              = 1  // 从帐号保留状态恢复了订阅
	SUBSCRIPTION_RENEWED                = 2  // 续订了处于活动状态的订阅
	SUBSCRIPTION_CANCELED               = 3  // 自愿或非自愿地取消了订阅。如果是自愿取消，在用户取消时发送
	SUBSCRIPTION_PURCHASED              = 4  // 购买了新的订阅
	SUBSCRIPTION_ON_HOLD                = 5  // 订阅已进入帐号保留状态（如果已启用）
	SUBSCRIPTION_IN_GRACE_PERIOD        = 6  // 订阅已进入宽限期（如果已启用）
	SUBSCRIPTION_RESTARTED              = 7  // 用户已通过 Play > 帐号 > 订阅重新激活其订阅（需要选择使用订阅恢复功能）
	SUBSCRIPTION_PRICE_CHANGE_CONFIRMED = 8  // 用户已成功确认订阅价格变动
	SUBSCRIPTION_DEFERRED               = 9  // 订阅的续订时间点已延期
	SUBSCRIPTION_PAUSED                 = 10 // 订阅已暂停
	SUBSCRIPTION_PAUSE_SCHEDULE_CHANGED = 11 // 订阅暂停计划已更改
	SUBSCRIPTION_REVOKED                = 12 // 用户在到期时间之前已撤消订阅
	SUBSCRIPTION_EXPIRED                = 13 // 订阅已到期
)

// google一次性购买的通知类型
const (
	ONE_TIME_PRODUCT_PURCHASED = 1 // 用户成功购买了一次性商品
	ONE_TIME_PRODUCT_CANCELED  = 2 // 用户已取消待处理的一次性商品购买交易
)

func PlayStoreNotifyType(v int) string {
	switch v {
	case SUBSCRIPTION_RECOVERED:
		return "SUBSCRIPTION_RECOVERED"
	case SUBSCRIPTION_RENEWED:
		return "SUBSCRIPTION_RENEWED"
	case SUBSCRIPTION_CANCELED:
		return "SUBSCRIPTION_CANCELED"
	case SUBSCRIPTION_PURCHASED:
		return "SUBSCRIPTION_PURCHASED"
	case SUBSCRIPTION_ON_HOLD:
		return "SUBSCRIPTION_ON_HOLD"
	case SUBSCRIPTION_IN_GRACE_PERIOD:
		return "SUBSCRIPTION_IN_GRACE_PERIOD"
	case SUBSCRIPTION_RESTARTED:
		return "SUBSCRIPTION_RESTARTED"
	case SUBSCRIPTION_PRICE_CHANGE_CONFIRMED:
		return "SUBSCRIPTION_PRICE_CHANGE_CONFIRMED"
	case SUBSCRIPTION_DEFERRED:
		return "SUBSCRIPTION_DEFERRED"
	case SUBSCRIPTION_PAUSED:
		return "SUBSCRIPTION_PAUSED"
	case SUBSCRIPTION_PAUSE_SCHEDULE_CHANGED:
		return "SUBSCRIPTION_PAUSE_SCHEDULE_CHANGED"
	case SUBSCRIPTION_REVOKED:
		return "SUBSCRIPTION_REVOKED"
	case SUBSCRIPTION_EXPIRED:
		return "SUBSCRIPTION_EXPIRED"
	}

	return ""
}

type PublisherService interface {
	playstore.IABProduct
	playstore.IABSubscription
}

func NewAndroidPublisherService(jsonkey []byte, client *http.Client) *playstore.Client {
	var err error
	var cli *playstore.Client

	if client == nil {
		cli, err = playstore.New(jsonkey)
	} else {
		cli, err = playstore.NewWithClient(jsonkey, client)
	}

	if err != nil {
		return nil
	}
	return cli
}

// RTDNotification 实时开发者通知
type RTDNotification struct {
	Subscription string `json:"subscription"`
	Message      struct {
		// Attributes map[string]interface{} `json:"attributes"`
		Base64Data string `json:"data"`
		MessageID  string `json:"messageId"`
	} `json:"message"`
}

func (noti *RTDNotification) GetDeveloperNotification() (*DeveloperNotification, error) {
	buf, err := base64.StdEncoding.DecodeString(noti.Message.Base64Data)
	if err != nil {
		return nil, err
	}

	var obj DeveloperNotification
	err = json.Unmarshal(buf, &obj)
	if err != nil {
		return nil, err
	}

	return &obj, nil
}

// https://developer.android.com/google/play/billing/rtdn-reference
type DeveloperNotification struct {
	Version                    string                     `json:"version"`
	PackageName                string                     `json:"packageName"`
	EventTimeMillis            string                     `json:"eventTimeMillis"`
	OneTimeProductNotification OneTimeProductNotification `json:"oneTimeProductNotification"`
	SubscriptionNotification   SubscriptionNotification   `json:"subscriptionNotification"`
	TestNotification           TestNotification           `json:"testNotification"`
}

// https://developer.android.com/google/play/billing/rtdn-reference
type OneTimeProductNotification struct {
	Version          string `json:"version"`
	NotificationType int    `json:"notificationType"`
	PurchaseToken    string `json:"purchaseToken"`
	Sku              string `json:"sku"`
}

// https://developer.android.com/google/play/billing/rtdn-reference
type SubscriptionNotification struct {
	Version          string `json:"version"`
	NotificationType int    `json:"notificationType"`
	PurchaseToken    string `json:"purchaseToken"`
	SubscriptionId   string `json:"subscriptionId"`
}

type TestNotification struct {
	Version string `json:"version"`
}
