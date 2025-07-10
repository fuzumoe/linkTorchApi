package repository_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
)

// mockMigrator implements the necessary AutoMigrate interface for testing.
type mockMigrator struct {
	calledWith []any
	errOn      any
}

func (m *mockMigrator) AutoMigrate(dst ...any) error {
	m.calledWith = append(m.calledWith, dst[0])
	if m.errOn != nil && reflect.TypeOf(dst[0]) == reflect.TypeOf(m.errOn) {
		// Return error with a message that will be wrapped by repository.Migrate.
		return fmt.Errorf("fail on %T", dst[0])
	}
	return nil
}

func TestMigrate(t *testing.T) {

	t.Run("Success", func(t *testing.T) {
		mm := &mockMigrator{}
		err := repository.Migrate(mm)
		assert.NoError(t, err)

		// Expect one call per model in model.AllModels.
		expected := model.AllModels
		assert.Equal(t, len(expected), len(mm.calledWith), "should call AutoMigrate for each model")

		// Verify order and types.
		for i, inst := range expected {
			assert.Equal(t, reflect.TypeOf(inst), reflect.TypeOf(mm.calledWith[i]),
				"call %d should migrate %T", i, inst)
		}
	})

	t.Run("Error", func(t *testing.T) {
		// Simulate error on migrating URL model.
		failModel := &model.URL{}
		mm := &mockMigrator{errOn: failModel}
		err := repository.Migrate(mm)
		assert.Error(t, err)
		// Check that error message contains both the auto-migrate prefix and our simulated error.
		assert.Contains(t, err.Error(), "auto-migrate", "error message should contain 'auto-migrate'")
		assert.Contains(t, err.Error(), "fail on *model.URL", "error should indicate failure on *model.URL")

		// Ensure migrator was called at least once.
		assert.Greater(t, len(mm.calledWith), 0, "should have attempted migrations before erroring")
	})
}
