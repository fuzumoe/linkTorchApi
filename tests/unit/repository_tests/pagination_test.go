package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fuzumoe/linkTorch-api/internal/repository"
)

func TestPaginationOffset(t *testing.T) {
	t.Run("Page 0 yields offset 0", func(t *testing.T) {
		p := repository.Pagination{
			Page:     0,
			PageSize: 20,
		}
		assert.Equal(t, 0, p.Offset(), "Page 0 should yield offset 0")
	})

	t.Run("Page 1 yields offset 0", func(t *testing.T) {
		p := repository.Pagination{
			Page:     1,
			PageSize: 20,
		}
		assert.Equal(t, 0, p.Offset(), "Page 1 should yield offset 0")
	})

	t.Run("Page 2 yields correct offset", func(t *testing.T) {
		p := repository.Pagination{
			Page:     2,
			PageSize: 20,
		}
		assert.Equal(t, 20, p.Offset(), "For Page 2 with PageSize 20, offset should be 20")
	})

	t.Run("Page 5 yields correct offset", func(t *testing.T) {
		p := repository.Pagination{
			Page:     5,
			PageSize: 10,
		}
		assert.Equal(t, 40, p.Offset(), "For Page 5 with PageSize 10, offset should be 40")
	})
}

func TestPaginationLimit(t *testing.T) {
	t.Run("Non-positive PageSize returns default limit", func(t *testing.T) {
		p1 := repository.Pagination{
			Page:     1,
			PageSize: 0,
		}
		assert.Equal(t, 10, p1.Limit(), "PageSize of 0 should return default limit of 10")

		p2 := repository.Pagination{
			Page:     1,
			PageSize: -5,
		}
		assert.Equal(t, 10, p2.Limit(), "Negative PageSize should return default limit of 10")
	})

	t.Run("Provided positive PageSize returns provided value", func(t *testing.T) {
		p := repository.Pagination{
			Page:     1,
			PageSize: 25,
		}
		assert.Equal(t, 25, p.Limit(), "Provided PageSize should be used as limit")
	})
}
