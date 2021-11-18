package unipay

import (
	"github.com/awa/go-iap/appstore"
)

// PlayStoreIAPRequest google play store iap
// https://developer.android.com/google/play/billing
type PlayStoreIAPRequest struct {
	PurchaseData     string `json:"purchase_data"`
	PurchaseDataSign string `json:"purchase_data_sign"`
}

type Request struct {
	// apple iap
	appstore.IAPRequest
	// google play store iap
	PlayStoreIAPRequest

	TransactionId string `json:"transaction_id"`

	// public
	PayWay    uint8  `json:"pay_way,string"` // 支付方式
	ProductID string `json:"goods_sn"`       // 商品编号,productID
	Timestamp string `json:"timestamp"`      // 请求时间戳
	Currency  string `json:"currency"`       // 货币单位
	Sign      string `json:"sign"`           // 请求签名

	Attach string `json:"-"` // 附件信息
}
