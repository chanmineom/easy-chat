package svc

import (
	"context"

	"github.com/chanmineom/easy-chat/apps/im/repository"
	"github.com/chanmineom/easy-chat/apps/im/rpc/internal/config"
)

type ServiceContext struct {
	Config config.Config
	Mongo  *repository.MongoStore
}

func NewServiceContext(ctx context.Context, c config.Config) (*ServiceContext, error) {
	mongoStore, err := repository.NewMongoStore(ctx, c.Mongo.Url, c.Mongo.Db)
	if err != nil {
		return nil, err
	}
	return &ServiceContext{
		Config: c,
		Mongo:  mongoStore,
	}, nil
}

func (s *ServiceContext) Close(ctx context.Context) error {
	return s.Mongo.Close(ctx)
}
