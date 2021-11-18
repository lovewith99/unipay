package unipay

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/androidpublisher/v3"
)

type GoogleOAuth2Svc interface {
	GetAccessToken() (*oauth2.Token, error)
}

type GoogleOAuth2 struct {
	Client *http.Client
	Config oauth2.Config

	State string
}

func (oa *GoogleOAuth2) GetAuthCodeUrl() string {
	return oa.Config.AuthCodeURL(
		oa.State,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)
}

func (oa *GoogleOAuth2) GetAuthCode() (string, error) {
	authcodeurl := oa.Config.AuthCodeURL(
		oa.State,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
	)

	resp, err := oa.Client.Get(authcodeurl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var authCode struct {
		Code  string `json:"code"`
		State string `json:"state"`
	}

	err = json.NewDecoder(resp.Body).Decode(&authCode)
	if err != nil {
		return "", err
	}

	return authCode.Code, nil
}

func (oa *GoogleOAuth2) GetAccessToken(code string) (*oauth2.Token, error) {
	ctx := context.WithValue(
		context.Background(),
		oauth2.HTTPClient, oa.Client,
	)

	return oa.Config.Exchange(ctx, code)
}

func (oa *GoogleOAuth2) RefreshToken(refreshToken string) (*oauth2.Token, error) {
	ctx := context.WithValue(
		context.Background(),
		oauth2.HTTPClient, oa.Client,
	)

	return oa.Config.Exchange(ctx, "",
		oauth2.SetAuthURLParam("grant_type", "refresh_token"),
		oauth2.SetAuthURLParam("refresh_token", refreshToken),
	)
}

type GoogleOauth2ConfigOption func(*GoogleOauth2Config)

type GoogleOauth2Config struct {
	sync.Mutex
	GoogleOAuth2

	token *oauth2.Token
}

// func Oauth2RedirectURL(uri string) GoogleOauth2ConfigOption {
// 	return func(cfg *GoogleOauth2Config) {
// 		cfg.GoogleOAuth2.Config.RedirectURL = uri
// 	}
// }

func Oauth2HttpClient(client *http.Client) GoogleOauth2ConfigOption {
	return func(cfg *GoogleOauth2Config) {
		cfg.GoogleOAuth2.Client = client
	}
}

func Oauth2RefreshToken(token string) GoogleOauth2ConfigOption {
	return func(cfg *GoogleOauth2Config) {
		cfg.token.RefreshToken = token
	}
}

func Oauth2State(state string) GoogleOauth2ConfigOption {
	return func(cfg *GoogleOauth2Config) {
		cfg.State = state
	}
}

func Oauth2Scopes(scopes ...string) GoogleOauth2ConfigOption {
	return func(cfg *GoogleOauth2Config) {
		cfg.GoogleOAuth2.Config.Scopes = scopes
	}
}

func NewGoogleOauth2Config(clientId, clientSecret, redirect string, opts ...GoogleOauth2ConfigOption) *GoogleOauth2Config {
	cfg := &GoogleOauth2Config{
		GoogleOAuth2: GoogleOAuth2{
			Config: oauth2.Config{
				ClientID:     clientId,
				ClientSecret: clientSecret,
				RedirectURL:  redirect,
				Endpoint:     google.Endpoint,
				Scopes: []string{
					androidpublisher.AndroidpublisherScope,
				},
			},
		},
		token: &oauth2.Token{},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.GoogleOAuth2.Client == nil {
		cfg.GoogleOAuth2.Client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return cfg
}

func (oa *GoogleOauth2Config) getFromCache() (*oauth2.Token, bool) {
	if oa.token.AccessToken != "" {
		if time.Now().Add(5 * time.Minute).Before(oa.token.Expiry) {
			// token 未过期
			return oa.token, true
		}
	}

	return nil, false
}

func (oa *GoogleOauth2Config) GetAccessToken() (*oauth2.Token, error) {
	if oa.State == "" {
		oa.State = "cn.leminet.oauth2"
	}

	if token, ok := oa.getFromCache(); ok {
		return token, nil
	}

	oa.Lock()
	defer oa.Unlock()

	if token, ok := oa.getFromCache(); ok {
		return token, nil
	}

	var err error
	var token *oauth2.Token
	cli := oa.GoogleOAuth2
	if len(oa.token.RefreshToken) > 0 {
		token, err = cli.RefreshToken(oa.token.RefreshToken)
	} else {
		if code, _ := cli.GetAuthCode(); code != "" {
			token, err = cli.GetAccessToken(code)
		} else {
			err = errors.New("get auth code failed")
		}
	}
	if err == nil {
		oa.token = token
	}

	return token, err
}
