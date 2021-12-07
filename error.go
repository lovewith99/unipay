package unipay

import "errors"

var (
	// 苹果/google订单不存在
	OrderNotFoundError = errors.New("transaction not found")
)

func IsOrderNotFondError(err error) bool {
	return err == OrderNotFoundError
}
