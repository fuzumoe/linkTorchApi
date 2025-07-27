package model_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/linkTorch-api/internal/model"
)

func TestAllModelsContainsExpectedTypes(t *testing.T) {
	// Expected model type names.
	expected := []string{
		"User",
		"URL",
		"AnalysisResult",
		"Link",
		"BlacklistedToken",
	}

	// Collect actual type names from model.AllModels.
	var actual []string
	for _, m := range model.AllModels {
		t := reflect.TypeOf(m)
		// m is a pointer to struct, so get element type.
		if t.Kind() == reflect.Ptr {
			actual = append(actual, t.Elem().Name())
		} else {
			actual = append(actual, t.Name())
		}
	}

	// Assert same length.
	assert.Len(t, actual, len(expected), "AllModels should contain %d entries", len(expected))

	// Assert each expected type is present.
	for _, name := range expected {
		assert.Contains(t, actual, name, "AllModels should include %s", name)
	}
}
