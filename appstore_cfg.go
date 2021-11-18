package unipay

import (
	"time"
)

type AppStoreClientConfig struct {
	bundleID string
	password string

	HttpTimeout time.Duration
}
