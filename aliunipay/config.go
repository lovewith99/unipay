package alipay

const (
	KeyMode  = "KeyMode"  // 普通公钥模式
	CertMode = "CertMode" // 公钥证书模式
)

type Config struct {
	IsProd bool
	Mode   string

	appId     string
	partnerId string

	privateKey string

	// 普通公钥
	aliPublicKey string

	// 公钥证书模式
	appCertSnFile       string // 应用公钥证书
	rootCertSnFile      string // 支付宝根证书
	aliPublicCertSnFile string // 支付宝公钥匙证书

	NotifyURL string
	ReturnURL string
}
