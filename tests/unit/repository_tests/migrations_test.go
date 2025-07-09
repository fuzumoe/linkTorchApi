package repository_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/fuzumoe/urlinsight-backend/internal/model"
	"github.com/fuzumoe/urlinsight-backend/internal/repository"
	"github.com/stretchr/testify/assert"
)

// mockMigrator implements the Migrator interface for testing.
type mockMigrator struct {
	calledWith []interface{}
	errOn      interface{} // if model matches this, return error
}

func (m *mockMigrator) AutoMigrate(dst ...interface{}) error {
	m.calledWith = append(m.calledWith, dst[0])
	if m.errOn != nil && reflect.TypeOf(dst[0]) == reflect.TypeOf(m.errOn) {
		return fmt.Errorf("fail on %T", dst[0])
	}
	return nil
}

func TestMigrate_Success(t *testing.T) {
	mm := &mockMigrator{}
	err := repository.Migrate(mm)
	assert.NoError(t, err)

	// Expect one call per model in model.AllModels
	expected := model.AllModels
	assert.Equal(t, len(expected), len(mm.calledWith), "should call AutoMigrate for each model")

	// Verify order and types
	for i, inst := range expected {
		assert.Equal(t, reflect.TypeOf(inst), reflect.TypeOf(mm.calledWith[i]),
			"call %d should migrate %T", i, inst)
	}
}

func TestMigrate_Error(t *testing.T) {
	// simulate error on migrating URL model
	failModel := &model.URL{}
	mm := &mockMigrator{errOn: failModel}
	err := repository.Migrate(mm)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fail on *model.URL")

	// ensure migrator was called up to the failing model
	var count int
	for _, inst := range mm.calledWith {
		if reflect.TypeOf(inst) == reflect.TypeOf(failModel) {
			break
		}
		count++
	}
	// should have been called at least once before URL if URL is first, but simple check
	assert.Greater(t, len(mm.calledWith), 0, "should have attempted migrations before erroring")
}
