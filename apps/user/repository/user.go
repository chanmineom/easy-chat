package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chanmineom/easy-chat/apps/user/model"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrPhoneRegistered = errors.New("phone is already registered")
)

func (s *MySQLStore) CreateUser(ctx context.Context, user model.User) error {
	_, err := s.conn.ExecCtx(
		ctx,
		`INSERT INTO users (
			id, phone, password_hash, nickname,
			avatar_url, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID,
		user.Phone,
		user.PasswordHash,
		user.Nickname,
		user.AvatarURL,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err == nil {
		return nil
	}

	var mysqlErr *mysqldriver.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return ErrPhoneRegistered
	}

	return fmt.Errorf("insert user: %w", err)
}

func (s *MySQLStore) FindByPhone(ctx context.Context, phone string) (model.User, error) {
	var user model.User

	err := s.conn.QueryRowCtx(
		ctx,
		&user,
		`SELECT id, phone, password_hash, nickname, avatar_url, status, created_at, updated_at
		 FROM users
		 WHERE phone = ?
		 LIMIT 1`,
		phone,
	)
	if errors.Is(err, sqlx.ErrNotFound) {
		return model.User{}, ErrUserNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("find user by phone: %w", err)
	}

	return user, nil
}

func (s *MySQLStore) FindByID(ctx context.Context, userID string) (model.User, error) {
	var user model.User

	err := s.conn.QueryRowCtx(
		ctx,
		&user,
		`SELECT id, phone, password_hash, nickname, avatar_url, status, created_at, updated_at
		 FROM users
		 WHERE id = ?
		 LIMIT 1`,
		userID,
	)
	if errors.Is(err, sqlx.ErrNotFound) {
		return model.User{}, ErrUserNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("find user by id: %w", err)
	}

	return user, nil
}

func (s *MySQLStore) TouchLastLogin(ctx context.Context, userID string, now time.Time) error {
	_, err := s.conn.ExecCtx(
		ctx,
		`UPDATE users
		 SET updated_at = ?
		 WHERE id = ?`,
		now,
		userID,
	)
	return err
}
