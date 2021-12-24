package unipay

type OrderInfo struct {
	Subject    string // 购买项目
	TotalFee   int    // 订单金额x100
	OutTradeNo string // 应用内交易流水号
	TradeNo    string // 第三方支付流水号
	Attach     string // 透传参数
	Currency   string // 货币单位, "CNY" | "USD"
}

type IOrder interface {
	Payed() bool
	OrderInfo() *OrderInfo
}

type OrderService interface {
	// Invoke 订单处理逻辑
	Invoke(order IOrder) error

	// Revoke 撤销订单, 执行与Invoke相反的逻辑
	Revoke(order IOrder) error

	// PostOrder 创建订单
	PostOrder(ctx *Context) (IOrder, error)

	// GetOrder 根据第三方交易流水号获取交易订单
	GetOrderByTradeNo(tradeno string, payway string) (IOrder, error)
}

type IapOrderService interface {
	OrderService
	CheckSubUser(ctx *Context, oriSubId, subId string) error
}

// Locker 订单锁, 防止并发处理同一笔订单导致而导致订单重复处理
type Locker interface {
	Lock(orderId string) (bool, error)
	UnLock(orderId string) error
}

// AttachService 保存/删除订单的附件信息
// 主要应用于apple iap 和 google iap场景下补单时, 订单的attach信息丢失
// 避免小票验证失败之后, 再次发起验证时, 订单的附件信息丢失, 无法正确处理回调
type AttachService interface {
	Create(orderId, attach string) error
	Delete(orderId string) error
}

// LockerImpl Locker的空实现
type LockerImpl struct{}

func (l LockerImpl) Lock(orderId string) (bool, error) {
	return true, nil
}

func (l LockerImpl) UnLock(orderId string) error {
	return nil
}

// AttachServiceImpl AttachService的空实现
type AttachServiceImpl struct{}

func (s AttachServiceImpl) Create(orderId, attach string) error {
	return nil
}

func (s AttachServiceImpl) Delete(orderId string) error {
	return nil
}
