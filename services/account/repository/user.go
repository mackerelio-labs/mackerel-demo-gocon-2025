package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/account/domain"
)

// UserRepository は domain.UserRepository に対するデータベースを使った実装
type UserRepository struct {
	db DB
}

func newUserRepository(db DB) *UserRepository {
	return &UserRepository{db}
}

// FIXME: simulateSomeUserSampling は一部のユーザーをサンプリングして重い計算処理を行う事をシミュレートする
func simulateSomeUserSampling(user *domain.User) {
	if user.ID%5 == 0 {
		time.Sleep(3 * time.Second)
	}
}

// Create は新規ユーザーを作成し, リポジトリに保存する
func (r *UserRepository) Create(ctx context.Context, input *domain.CreateUserInput) (*domain.User, error) {
	id, err := generateID(r.db)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		ID:           domain.UserID(id),
		Name:         input.Name,
		PasswordHash: input.PasswordHash,
	}
	_, err = r.db.ExecContext(
		ctx,
		`
			INSERT INTO users (id, name, password_hash)
				VALUES (?, ?, ?)
		`,
		user.ID, user.Name, user.PasswordHash,
	)
	if err != nil {
		return nil, err
	}
	simulateSomeUserSampling(user)
	return user, nil
}

// FindByID はリポジトリから ID でユーザーを検索する
func (r *UserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	var user domain.User
	err := sqlx.GetContext(
		ctx,
		r.db,
		&user,
		`
			SELECT id, name, password_hash FROM users
				WHERE id = ? LIMIT 1
		`,
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	simulateSomeUserSampling(&user)
	return &user, nil
}

// FindByName はリポジトリから名前でユーザーを検索する
func (r *UserRepository) FindByName(ctx context.Context, name string) (*domain.User, error) {
	var user domain.User
	err := sqlx.GetContext(
		ctx,
		r.db,
		&user,
		`
			SELECT id, name, password_hash FROM users
				WHERE name = ? LIMIT 1
		`,
		name,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	simulateSomeUserSampling(&user)
	return &user, nil
}
