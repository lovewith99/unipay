package unipay

type UniPayOrderInfo struct {
	Subject    string // 购买项目
	TotalFee   int    // 订单金额x100
	OutTradeNo string // 应用内交易流水号
	TradeNo    string // 第三方支付流水号
	Attach     string // 透传参数
	Currency   string // 货币单位, "CNY" | "USD"
}

type UniPayOrder interface {
	Payed() bool
}

type UniPayOrderService interface {
	// PayCB 支付回调处理
	PayCB(order UniPayOrder) error

	// Revoke 撤销订单, 执行与PayCB相反的逻辑
	Revoke(order UniPayOrder) error

	// PostOrder 创建订单
	PostOrder(ctx *Context) (UniPayOrder, error)

	// GetOrder 根据第三方交易流水号获取交易订单
	GetOrderByTradeNo(tradeno string, payway string) (UniPayOrder, error)
}

type IapOrderService interface {
	UniPayOrderService
	CheckSubUser(ctx *Context, oriSubId, subId string) bool
}

// OrderLocker 订单锁, 防止并发处理同一笔订单导致而导致订单重复处理
type OrderLocker interface {
	Lock(orderId string) (bool, error)
	UnLock(orderId string) error
}

type EmptyOrderLocker struct{}

func (l EmptyOrderLocker) Lock(orderId string) (bool, error) {
	return true, nil
}

func (l EmptyOrderLocker) UnLock(orderId string) error {
	return nil
}

// OrderAttachSvc 保存/删除订单的附件信息
// 主要应用于apple iap 和 google iap场景下补单时, 订单的attach信息丢失
// 避免小票验证失败之后, 再次发起验证时, 订单的附件信息丢失, 无法正确处理回调
type OrderAttachService interface {
	Create(orderId, attach string) error
	Delete(orderId string) error
}

type EmptyOrderAttachService struct{}

func (s EmptyOrderAttachService) Create(orderId, attach string) error {
	return nil
}

func (s EmptyOrderAttachService) Delete(orderId string) error {
	return nil
}
