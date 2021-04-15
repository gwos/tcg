package cache

import (
	"github.com/patrickmn/go-cache"
	"time"
)

// AuthCache used by Controller.validateToken
var AuthCache = cache.New(8*time.Hour, time.Hour)

// TraceTokenCache used by AgentService
var TraceTokenCache = cache.New(-1, -1)

// ProcessesCache used by Server Connector
var ProcessesCache = cache.New(5*time.Minute, 5*time.Minute)

// Credentials defines type of AuthCache items
type Credentials struct {
	GwosAppName  string
	GwosAPIToken string
}
