package cache

import (
	"github.com/huoxue1/td-onebot/internal/cache/impl"
	"github.com/huoxue1/td-onebot/internal/conf"
	"time"
)

type Client interface {
	Set(key, value string) error
	SetTtl(key, value string, ttl time.Duration) error
	Get(key string) string
	GetDefault(key string, defaultValue string) string
	Delete(key string) error
	ForEach(func(key, value string) bool)

	SetAdd(key, value string) error
	SetIsMem(key string, member string) bool
	SetDel(key string, member string) error
}

var (
	c Client
)

func GetCache() Client {
	return c
}

func InitCache(config *conf.Config) error {
	if config.Cache.CacheType == "redis" {
		client, err := impl.InitRedis(config)
		if err != nil {
			return err
		}

		c = client
	} else {
		client, err := impl.InitNustdb(config)
		if err != nil {
			return err
		}
		c = client
	}
	return nil
}
