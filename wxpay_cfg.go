package unipay

type WxPayClientConfig struct {
	appId    string
	mchId    string
	key      string
	certPem  string
	keyPem   string
	signType string

	NotifyURL string
}
