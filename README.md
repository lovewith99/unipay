# unipay

## 介绍
unipay 抽象了服务端处理app内支付的过程和逻辑, 将具体的业务处理操作抽象成下面几个接口。

**UniPayOrder**
```golang
// 描述app内订单的接口
type UniPayOrder interface {
	Payed() bool // 订单是否已支付
}
```

**UniPayOrderService**
```golang
// 一个通用的订单处理接口
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
```

**IapOrderService**
```golang
// app的账号系统通常独立于appstore和playstore
// 有时候在处理"订阅"相关的订单时, 需要校验app当前登入账号和订阅发起者之间的关系 
// 这时候就需要实现IapOrderService中的CheckSubUser方法
type IapOrderService interface {
	UniPayOrderService
	CheckSubUser(ctx *Context, oriSubId, subId string) bool
}
```

**OrderLocker**
``` golang
// 该接口并不是一个必须要实现的接口
// OrderLocker 订单锁, 防止并发处理同一笔订单导致而导致订单重复处理
type OrderLocker interface {
	Lock(orderId string) (bool, error)
	UnLock(orderId string) error
}
```

**OrderAttachService**
```golang
// 该接口并不是一个必须要实现的接口
// OrderAttachSvc 保存/删除订单的附件信息
// 主要应用于apple iap 和 google iap场景下补单时, 订单的attach信息丢失
// 避免小票验证失败之后, 再次发起验证时, 订单的附件信息丢失, 无法正确处理回调
type OrderAttachService interface {
	Create(orderId, attach string) error
	Delete(orderId string) error
}
```

**将UniPayOrder转换成UniPayOrderInfo**
> func(UniPayOrder) *UniPayOrderInfo
```golang
// 该方法签名定义了将一个UniPayOrder转换成UniPayOrderInfo的函数
// 主要用在alipay, wxpay, paypal 支付场景下生成支付链接时获取订单信息
// 用来描述订单的基本信息
type UniPayOrderInfo struct {
	Subject    string // 购买项目
	TotalFee   int    // 订单金额x100
	OutTradeNo string // 应用内交易流水号
	TradeNo    string // 第三方支付流水号
	Attach     string // 透传参数
	Currency   string // 货币单位, "CNY" | "USD"
}
```

## apple store

### 初始化
```golang
applepay = unipay.NewAppStoreClient(
	"password",
	"bundleID",
	unipay.AppStoreOrderLocker(Interface<OrderLocker>),
	unipay.AppStoreAttachSvc(Interface<OrderAttachService>),
	unipay.AppStoreOrderSvc(Interface<IapOrderService>),
)
```


## play store
### 初始化
```golang
client, err := unipay.NewPlayStoreClient(
	unipay.PlayStorePackageName("packagename"),
	unipay.PlayStoreOrderLocker(Interface<>),
	unipay.PlayStoreAttachSvc(Interface<>),
	unipay.PlayStoreOrderSvc(Interface<>)
	unipay.PlayStorePublicKey("publickey"), // todo
	unipay.PlayStoreAndroidPublisherSvc(Interface<>),
)
```


## paypal v2


## alipay v3


## wxpay 