package repository_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/linkTorch-api/internal/model"
	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

type mockMigrator struct {
	calledWith []any
	errOn      any
}

func (m *mockMigrator) AutoMigrate(dst ...any) error {
	m.calledWith = append(m.calledWith, dst[0])
	if m.errOn != nil && reflect.TypeOf(dst[0]) == reflect.TypeOf(m.errOn) {
		return fmt.Errorf("fail on %T", dst[0])
	}
	return nil
}

func TestMigrate(t *testing.T) {

	t.Run("Success", func(t *testing.T) {
		mm := &mockMigrator{}
		err := repository.Migrate(mm)
		assert.NoError(t, err)

		expected := model.AllModels
		assert.Equal(t, len(expected), len(mm.calledWith), "should call AutoMigrate for each model")

		for i, inst := range expected {
			assert.Equal(t, reflect.TypeOf(inst), reflect.TypeOf(mm.calledWith[i]),
				"call %d should migrate %T", i, inst)
		}
	})

	t.Run("Error", func(t *testing.T) {
		failModel := &model.URL{}
		mm := &mockMigrator{errOn: failModel}
		err := repository.Migrate(mm)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "auto-migrate", "error message should contain 'auto-migrate'")
		assert.Contains(t, err.Error(), "fail on *model.URL", "error should indicate failure on *model.URL")

		assert.Greater(t, len(mm.calledWith), 0, "should have attempted migrations before erroring")
	})
}
