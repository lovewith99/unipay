package playunipay

type Config struct {
	PackageName string

	// jsonKey   []byte
	publicKey string

	// 获取AccessToken
	// clientId     string
	// clientSecret string
	// refreshToken string
	// RedirectURL  string // OAuth2 获取AccessToken 授权重定向地址
}
