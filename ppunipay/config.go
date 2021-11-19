package ppunipay

type Config struct {
	IsProd   bool
	clientId string
	secret   string

	ReturnURL string
	CancelURL string
}
