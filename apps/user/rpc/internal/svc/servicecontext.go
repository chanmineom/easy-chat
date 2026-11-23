package svc

import (
	"github.com/chanmineom/easy-chat/apps/user/repository"
	"github.com/chanmineom/easy-chat/apps/user/rpc/internal/config"
)

type ServiceContext struct {
	Config config.Config
	Store  *repository.MySQLStore
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		Store:  repository.NewMySQLStore(c.Mysql.DataSource),
	}
}
