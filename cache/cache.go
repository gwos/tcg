package cache

import (
	"github.com/patrickmn/go-cache"
	"time"
)

var AuthCache = cache.New(8*time.Hour, time.Hour)

var NatsCache = cache.New(10*time.Minute, 10*time.Minute)

type Credentials struct {
	GwosAppName  string
	GwosAPIToken string
}
