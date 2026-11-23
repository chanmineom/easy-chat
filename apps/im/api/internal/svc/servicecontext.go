// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package svc

import (
	"time"

	"github.com/chanmineom/easy-chat/apps/im/api/internal/config"
	"github.com/chanmineom/easy-chat/apps/im/rpc/imclient"
	"github.com/chanmineom/easy-chat/apps/im/ticket"
	"github.com/chanmineom/easy-chat/apps/social/rpc/socialclient"
	"github.com/chanmineom/easy-chat/apps/user/rpc/userclient"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config  config.Config
	Im      imclient.Im
	Social  socialclient.Social
	User    userclient.User
	Tickets ticket.Registry
}

func NewServiceContext(c config.Config) *ServiceContext {
	redisClient := redis.MustNewRedis(c.Redis)

	return &ServiceContext{
		Config: c,
		Im:     imclient.NewIm(zrpc.MustNewClient(c.ImRpc)),
		Social: socialclient.NewSocial(zrpc.MustNewClient(c.SocialRpc)),
		User:   userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
		Tickets: ticket.NewRedisRegistry(
			redisClient,
			30*time.Second,
		),
	}
}
