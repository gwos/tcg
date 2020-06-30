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
var DispatcherRetryCache = cache.New(30*time.Minute, 30*time.Minute)

// DispatcherWorkersCache used by NATS Dispatcher
var DispatcherWorkersCache = cache.New(-1, -1)

// TraceTokenCache used by AgentService
var TraceTokenCache = cache.New(-1, -1)

// ProcessesCache used by Server Connector
var ProcessesCache = cache.New(5*time.Minute, 5*time.Minute)

// Credentials defines type of AuthCache items
type Credentials struct {
	GwosAppName  string
	GwosAPIToken string
}

// Licenses Cache
// licence check result is needed only within one sync run, when next sync it must be refreshed so 5 mins OK
var LicenseChecksCache = cache.New(5*time.Minute, 5*time.Minute)

// prev sent hosts count is needed only within one sync run (to revert last sent hosts count to prev values
// if sync failed and inventory was not sent)
var PrevSentHostsCountCache = cache.New(5*time.Minute, 5*time.Minute)

// need to keep last hosts sent count and they must be shared between curr and next sync runs
// and we cannot know an interval between two sync runs
var LastSentHostsCountCache = cache.New(cache.NoExpiration, cache.NoExpiration)
