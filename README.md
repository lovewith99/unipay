# unipay

## 介绍
unipay 抽象了服务端处理app内支付的过程和逻辑, 将具体的业务处理操作抽象成下面几个接口。

**IOrder**
```golang
// 描述app内订单的接口
type IOrder interface {
	Payed() bool // 订单是否已支付
	OrderInfo() *OrderInfo // 获取订单的基本信息
}
```

**OrderService**
```golang
// 一个通用的订单处理接口
type OrderService interface {
	// Invoke 订单处理逻辑
	Invoke(order UniPayOrder) error

	// Revoke 撤销订单, 执行与Invoke相反的逻辑
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

**Locker**
``` golang
// 该接口并不是一个必须要实现的接口
// OrderLocker 订单锁, 防止并发处理同一笔订单导致而导致订单重复处理
type Locker interface {
	Lock(orderId string) (bool, error)
	UnLock(orderId string) error
}
```

**AttachService**
```golang
// 该接口并不是一个必须要实现的接口
// AttachSvc 保存/删除订单的附件信息
// 主要应用于apple iap 和 google iap场景下补单时, 订单的attach信息丢失
// 避免小票验证失败之后, 再次发起验证时, 订单的附件信息丢失, 无法正确处理回调
type AttachService interface {
	Create(orderId, attach string) error
	Delete(orderId string) error
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
### 初始化
```golang
client, _ := unipay.NewPayPalClient(
		"clientId",
		"secret",
		true, // 生产环境: true, 沙盒环境: false
		unipay.PayPalOrderSvc(Interface<UniPayOrderService>),
		unipay.PayPalNotifyURL("ReturnURL", "CancelURL"),
		unipay.PayPalGetOrderInfoFunc(f func(UniPayOrder) *UniPayOrderInfo),
)
```

## alipay v3
### 初始化
```golang
// 普通公钥模式: KeyMode
alipay, _ = unipay.NewAliPayClient(
		unipay.AliPayEnv(true), // 生产环境:true, 沙盒环境: false
		unipay.AliPayConfig("appId", "partnerId"),
		unipay.AliPayPrivateKey(""),
		unipay.AliPayAliPublicKey(""),
		unipay.AliPayNotifyURL("NotifyURL"),
		unipay.AliPayReturnURL("ReturnURL"),
		unipay.AliPayOrderSvc(Interface<UniPayOrderService>),
		unipay.AliPayGetOrderInfoFunc(f func(UniPayOrder) *UniPayOrderInfo),
)
// 证书模式: CertMode 
alipay, _ = unipay.NewAliPayClient(
		unipay.AliPayEnv(true), // 生产环境:true, 沙盒环境: false
		unipay.AliPayMode("CertMode"),
		unipay.AliPayConfig("appId", "partnerId"),
		unipay.AliPayCertFile("appCertSn", "rootCertSn", "aliPublicCertSn"),
		unipay.AliPayNotifyURL("NotifyURL"),
		unipay.AliPayReturnURL("ReturnURL"),
		unipay.AliPayOrderSvc(Interface<UniPayOrderService>),
		unipay.AliPayGetOrderInfoFunc(f func(UniPayOrder) *UniPayOrderInfo),
)
```



## wxpay 

### 初始化
```golang
client, _ := unipay.NewWxPayClient(
	unipay.WxPayOrderSvc(Interface<UniPayOrderService>),
	unipay.WxPayNotifyURL("NotifyURL"),
	unipay.WxPayConfig("appId", "mchId", "key"),
	unipay.WxPayGetOrderInfoFunc(f func(UniPayOrder) *UniPayOrderInfo),
)
```