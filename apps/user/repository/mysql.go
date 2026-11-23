package repository

import "github.com/zeromicro/go-zero/core/stores/sqlx"

type MySQLStore struct {
	conn sqlx.SqlConn
}

func NewMySQLStore(dataSource string) *MySQLStore {
	return &MySQLStore{
		conn: sqlx.NewMysql(dataSource),
	}
}
