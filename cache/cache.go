package cache

import (
	"github.com/patrickmn/go-cache"
	"time"
)

// AuthCache used by Controller.validateToken
var AuthCache = cache.New(8*time.Hour, time.Hour)

// DispatcherDoneCache used by NATS Dispatcher
var DispatcherDoneCache = cache.New(10*time.Minute, 10*time.Minute)

// DispatcherRetryCache used by NATS Dispatcher
var DispatcherRetryCache = cache.New(22*time.Minute, 22*time.Minute)

// TraceTokenCache used by AgentService
var TraceTokenCache = cache.New(-1, -1)

// Credentials defines type of AuthCache items
type Credentials struct {
	GwosAppName  string
	GwosAPIToken string
}
