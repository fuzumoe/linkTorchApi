package model_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

func TestAllModelsContainsExpectedTypes(t *testing.T) {
	expected := []string{
		"User",
		"URL",
		"AnalysisResult",
		"Link",
		"BlacklistedToken",
	}

	var actual []string
	for _, m := range model.AllModels {
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Ptr {
			actual = append(actual, t.Elem().Name())
		} else {
			actual = append(actual, t.Name())
		}
	}

	assert.Len(t, actual, len(expected), "AllModels should contain %d entries", len(expected))

	for _, name := range expected {
		assert.Contains(t, actual, name, "AllModels should include %s", name)
	}
}
