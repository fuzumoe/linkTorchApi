package repository_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

func setupUserMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

func TestUserRepository(t *testing.T) {
	fixedTime := time.Now()

	t.Run("Create", func(t *testing.T) {
		db, mock := setupUserMockDB(t)
		repo := repository.NewUserRepo(db)
		user := &model.User{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "hashedpassword",
			Role:     "user",
		}

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"INSERT INTO `users` (`username`,`email`,`password`,`role`,`created_at`,`updated_at`,`deleted_at`) VALUES (?,?,?,?,?,?,?)",
		)).WithArgs(
			user.Username,
			user.Email,
			user.Password,
			user.Role,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.Create(user)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByID", func(t *testing.T) {
		db, mock := setupUserMockDB(t)
		repo := repository.NewUserRepo(db)
		userID := uint(1)

		rows := sqlmock.NewRows([]string{"id", "username", "email", "password", "created_at", "updated_at", "deleted_at"}).
			AddRow(userID, "testuser", "test@example.com", "hashedpassword", fixedTime, fixedTime, nil)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `users` WHERE `users`.`id` = ? AND `users`.`deleted_at` IS NULL ORDER BY `users`.`id` LIMIT ?",
		)).WithArgs(userID, 1).WillReturnRows(rows)

		user, err := repo.FindByID(userID)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "test@example.com", user.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByID Not Found", func(t *testing.T) {
		db, mock := setupUserMockDB(t)
		repo := repository.NewUserRepo(db)
		userID := uint(999)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `users` WHERE `users`.`id` = ? AND `users`.`deleted_at` IS NULL ORDER BY `users`.`id` LIMIT ?",
		)).WithArgs(userID, 1).WillReturnError(gorm.ErrRecordNotFound)

		user, err := repo.FindByID(userID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByEmail", func(t *testing.T) {
		db, mock := setupUserMockDB(t)
		repo := repository.NewUserRepo(db)
		email := "test@example.com"

		rows := sqlmock.NewRows([]string{"id", "username", "email", "password", "created_at", "updated_at", "deleted_at"}).
			AddRow(1, "testuser", email, "hashedpassword", fixedTime, fixedTime, nil)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `users` WHERE email = ? AND `users`.`deleted_at` IS NULL ORDER BY `users`.`id` LIMIT ?",
		)).WithArgs(email, 1).WillReturnRows(rows)

		user, err := repo.FindByEmail(email)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, email, user.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FindByEmail Not Found", func(t *testing.T) {
		db, mock := setupUserMockDB(t)
		repo := repository.NewUserRepo(db)
		email := "nonexistent@example.com"

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `users` WHERE email = ? AND `users`.`deleted_at` IS NULL ORDER BY `users`.`id` LIMIT ?",
		)).WithArgs(email, 1).WillReturnError(gorm.ErrRecordNotFound)

		user, err := repo.FindByEmail(email)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Search", func(t *testing.T) {
		db, mock := setupUserMockDB(t)
		repo := repository.NewUserRepo(db)
		pagination := repository.Pagination{Page: 1, PageSize: 10}

		rows := sqlmock.NewRows([]string{"id", "username", "email", "password", "created_at", "updated_at", "deleted_at"}).
			AddRow(1, "user1", "user1@example.com", "hash1", fixedTime, fixedTime, nil).
			AddRow(2, "user2", "user2@example.com", "hash2", fixedTime, fixedTime, nil)

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `users` WHERE `users`.`deleted_at` IS NULL LIMIT ?",
		)).WithArgs(10).WillReturnRows(rows)

		users, err := repo.Search("", "", "", pagination)
		assert.NoError(t, err)
		assert.Len(t, users, 2)
		assert.Equal(t, "user1", users[0].Username)
		assert.Equal(t, "user2", users[1].Username)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ListAll Empty", func(t *testing.T) {
		db, mock := setupUserMockDB(t)
		repo := repository.NewUserRepo(db)
		pagination := repository.Pagination{Page: 1, PageSize: 10}

		rows := sqlmock.NewRows([]string{"id", "username", "email", "password", "created_at", "updated_at", "deleted_at"})

		mock.ExpectQuery(regexp.QuoteMeta(
			"SELECT * FROM `users` WHERE `users`.`deleted_at` IS NULL LIMIT ?",
		)).WithArgs(10).WillReturnRows(rows)

		users, err := repo.Search("", "", "", pagination)
		assert.NoError(t, err)
		assert.Empty(t, users)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete", func(t *testing.T) {
		db, mock := setupUserMockDB(t)
		repo := repository.NewUserRepo(db)
		userID := uint(1)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `users` SET `deleted_at`=? WHERE `users`.`id` = ? AND `users`.`deleted_at` IS NULL",
		)).WithArgs(sqlmock.AnyArg(), userID).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.Delete(userID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete Not Found", func(t *testing.T) {
		db, mock := setupUserMockDB(t)
		repo := repository.NewUserRepo(db)
		userID := uint(999)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			"UPDATE `users` SET `deleted_at`=? WHERE `users`.`id` = ? AND `users`.`deleted_at` IS NULL",
		)).WithArgs(sqlmock.AnyArg(), userID).WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectCommit()

		err := repo.Delete(userID)
		assert.Error(t, err)
		assert.Equal(t, "user not found", err.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
