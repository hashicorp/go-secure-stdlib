package regexp

import (
	"github.com/jellydator/ttlcache/v3"
	"regexp"
	"sync"
	"time"
)

// Caches regexp compilation to avoid CPU and RAM usage for many duplicate regexps

const defaultTTL = 2 * time.Minute

var (
	regexpCache      *ttlcache.Cache[string, *regexp.Regexp]
	posixRegexpCache *ttlcache.Cache[string, *regexp.Regexp]
	ticker           *time.Ticker
	stopchan         chan struct{}
	setupLock        sync.Mutex
)

func init() {
	SetCompileCacheTTL(defaultTTL)
}

func SetCompileCacheTTL(ttl time.Duration) {
	Stop()
	setupLock.Lock()
	defer setupLock.Unlock()

	regexpCache = ttlcache.New[string, *regexp.Regexp](ttlcache.WithTTL[string, *regexp.Regexp](ttl))
	posixRegexpCache = ttlcache.New[string, *regexp.Regexp](ttlcache.WithTTL[string, *regexp.Regexp](ttl))
	ticker = time.NewTicker(ttl)
	stopchan = make(chan struct{})

	go expire()
}

func expire() {
	for {
		select {
		case <-ticker.C:
			regexpCache.DeleteExpired()
			posixRegexpCache.DeleteExpired()
		case <-stopchan:
			ticker.Stop()
			break
		}
	}
}
func MustCompile(pattern string) *regexp.Regexp {
	item := regexpCache.Get(pattern)
	if item != nil {
		return item.Value()
	}
	regex := regexp.MustCompile(pattern)
	regexpCache.Set(pattern, regex, ttlcache.DefaultTTL)
	return regex
}

func MustCompilePOSIX(pattern string) *regexp.Regexp {
	item := posixRegexpCache.Get(pattern)
	if item != nil {
		return item.Value()
	}
	regex := regexp.MustCompilePOSIX(pattern)
	posixRegexpCache.Set(pattern, regex, ttlcache.DefaultTTL)
	return regex
}

func Compile(pattern string) (*regexp.Regexp, error) {
	item := regexpCache.Get(pattern)
	if item != nil {
		return item.Value(), nil
	}
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	regexpCache.Set(pattern, regex, ttlcache.DefaultTTL)
	return regex, nil
}

func CompilePOSIX(pattern string) (*regexp.Regexp, error) {
	item := posixRegexpCache.Get(pattern)
	if item != nil {
		return item.Value(), nil
	}
	regex, err := regexp.CompilePOSIX(pattern)
	if err != nil {
		return nil, err
	}
	posixRegexpCache.Set(pattern, regex, ttlcache.DefaultTTL)
	return regex, nil
}

func Stop() {
	setupLock.Lock()
	defer setupLock.Unlock()
	if ticker != nil {
		stopchan <- struct{}{}
	}
}
