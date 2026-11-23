package svc

import (
	"github.com/chanmineom/easy-chat/apps/im/rpc/imclient"
	"github.com/chanmineom/easy-chat/apps/social/outbox"
	"github.com/chanmineom/easy-chat/apps/social/repository"
	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/config"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	Store        *repository.MySQLStore
	ImRpc        imclient.Im
	OutboxWorker *outbox.Worker
}

func NewServiceContext(c config.Config) *ServiceContext {
	store := repository.NewMySQLStore(c.Mysql.Datasource)
	imRpc := imclient.NewIm(zrpc.MustNewClient(c.ImRpc))

	return &ServiceContext{
		Config:       c,
		Store:        store,
		ImRpc:        imRpc,
		OutboxWorker: outbox.NewWorker(store, imRpc),
	}
}
