package repository_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

func setupUserRepoMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	// Create sqlmock DB
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	// Create a GORM DB using the mock connection.
	dialector := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

func TestUserRepo(t *testing.T) {
	t.Run("Create Success", func(t *testing.T) {
		db, mock := setupUserRepoMock(t)
		repo := repository.NewUserRepo(db)

		user := &model.User{Username: "alice", Email: "a@b.com", Password: "pass"}

		// Expectation: INSERT INTO `users`
		mock.ExpectBegin()
		// Use AnyArg() for dynamic fields like timestamps and IDs.
		mock.ExpectExec("INSERT INTO `users`").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.Create(user)
		assert.NoError(t, err)
	})

	t.Run("FindByID NotFound", func(t *testing.T) {
		db, mock := setupUserRepoMock(t)
		repo := repository.NewUserRepo(db)

		// Expect query with soft delete condition and LIMIT 1.
		mock.ExpectQuery("SELECT .* FROM `users` WHERE .*`id` = .* AND .*`deleted_at` IS NULL.*").
			WithArgs(42, 1). // GORM passes LIMIT as a separate arg.
			WillReturnError(gorm.ErrRecordNotFound)

		u, err := repo.FindByID(42)
		assert.Nil(t, u)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("FindByEmail Success", func(t *testing.T) {
		db, mock := setupUserRepoMock(t)
		repo := repository.NewUserRepo(db)

		example := &model.User{ID: 7, Username: "bob", Email: "bob@c.com"}
		rows := sqlmock.NewRows([]string{"id", "username", "email", "password", "created_at", "updated_at", "deleted_at"}).
			AddRow(example.ID, example.Username, example.Email, example.Password, example.CreatedAt, example.UpdatedAt, nil)

		// Expect query including soft delete condition and LIMIT param.
		mock.ExpectQuery("SELECT .* FROM `users` WHERE email = .* AND .*`deleted_at` IS NULL.*").
			WithArgs(example.Email, 1).
			WillReturnRows(rows)

		user, err := repo.FindByEmail(example.Email)
		assert.NoError(t, err)
		assert.Equal(t, example.ID, user.ID)
		assert.Equal(t, example.Email, user.Email)
	})

	t.Run("ListAll Success", func(t *testing.T) {
		db, mock := setupUserRepoMock(t)
		repo := repository.NewUserRepo(db)

		rows := sqlmock.NewRows([]string{"id", "username", "email", "password", "created_at", "updated_at", "deleted_at"}).
			AddRow(1, "u1", "a@b", "p", nil, nil, nil).
			AddRow(2, "u2", "c@d", "q", nil, nil, nil)

		// Expect query with soft delete condition.
		mock.ExpectQuery("SELECT .* FROM `users` WHERE .*`deleted_at` IS NULL.*").
			WillReturnRows(rows)

		users, err := repo.ListAll()
		assert.NoError(t, err)
		assert.Len(t, users, 2)
	})

	t.Run("Delete NotFound", func(t *testing.T) {
		db, mock := setupUserRepoMock(t)
		repo := repository.NewUserRepo(db)

		// GORM wraps the delete (soft delete) in a transaction.
		mock.ExpectBegin()
		// For soft delete, GORM issues an UPDATE on the `deleted_at` field.
		mock.ExpectExec("UPDATE `users` SET .*`deleted_at`.*WHERE .*`id` = .*").
			WithArgs(sqlmock.AnyArg(), 100). // First arg is timestamp, second is user ID.
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		err := repo.Delete(100)
		assert.EqualError(t, err, "user not found")
	})

	t.Run("Delete Success", func(t *testing.T) {
		db, mock := setupUserRepoMock(t)
		repo := repository.NewUserRepo(db)

		// GORM wraps the delete (soft delete) in a transaction.
		mock.ExpectBegin()
		// For soft delete, it's an UPDATE on the `deleted_at` field.
		mock.ExpectExec("UPDATE `users` SET .*`deleted_at`.*WHERE .*`id` = .*").
			WithArgs(sqlmock.AnyArg(), 5). // First arg is timestamp, second is user ID.
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.Delete(5)
		assert.NoError(t, err)
	})
}
