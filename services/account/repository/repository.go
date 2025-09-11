package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/mackerelio-labs/mackerel-demo-gocon-2025/services/account/domain"
)

// DB はデータベースのインターフェース
type DB interface {
	sqlx.Execer
	sqlx.ExecerContext
	sqlx.Queryer
	sqlx.QueryerContext
}

// Repository は domain.Repository に対するデータベースを使った実装
type Repository struct {
	user *UserRepository
}

// NewRepository は Repository を作成する
func NewRepository(db DB) *Repository {
	return &Repository{
		user: newUserRepository(db),
	}
}

// User はユーザーに対するリポジトリを返す
func (r *Repository) User() domain.UserRepository {
	return r.user
}

func generateID(db DB) (uint64, error) {
	var id uint64
	err := sqlx.Get(db, &id, "SELECT UUID_SHORT()")
	return id, err
}
