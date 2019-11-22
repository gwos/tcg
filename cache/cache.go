package cache

import (
    "github.com/patrickmn/go-cache"
    "time"
)

var AuthCache = cache.New(8*time.Hour, time.Hour)

type Credentials struct {
    GwosAppName  string
    GwosApiToken string
}
