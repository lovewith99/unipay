package uniapple

import (
	"time"
)

type Config struct {
	bundleID string
	password string

	HttpTimeout time.Duration
}
