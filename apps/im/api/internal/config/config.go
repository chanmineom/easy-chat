// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf

	ImRpc     zrpc.RpcClientConf
	SocialRpc zrpc.RpcClientConf
	UserRpc   zrpc.RpcClientConf

	JwtAuth struct {
		AccessSecret string
	}

	Redis redis.RedisConf
}
