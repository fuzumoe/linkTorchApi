package analyzer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"

	"github.com/fuzumoe/linkTorch-api/internal/analyzer"
	"github.com/fuzumoe/linkTorch-api/internal/model"
)

func TestLinkChecker_Run(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			}
		case "/get":
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("GET OK"))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	t.Run("Setup and Parse Base URL", func(t *testing.T) {
		_, err := url.Parse(ts.URL)
		require.NoError(t, err)
	})

	lc := analyzer.NewLinkChecker(2, 2*time.Second)
	require.NotNil(t, lc)

	lcValue := reflect.ValueOf(lc).Elem()
	clientField := lcValue.FieldByName("client")
	require.True(t, clientField.IsValid(), "client field must be valid")
	ptrToClient := unsafe.Pointer(clientField.UnsafeAddr())
	reflect.NewAt(clientField.Type(), ptrToClient).Elem().Set(reflect.ValueOf(ts.Client()))

	link1 := model.Link{Href: ts.URL + "/ok"}
	link2 := model.Link{Href: ts.URL + "/get"}
	links := []model.Link{link1, link2}

	t.Run("Run LinkChecker", func(t *testing.T) {
		updatedLinks := lc.Run(context.Background(), links)
		require.Len(t, updatedLinks, 2, "Expected two links returned by Run()")

		t.Run("Verify All Links Have Status OK", func(t *testing.T) {
			for _, link := range updatedLinks {
				require.Equal(t, http.StatusOK, link.StatusCode, "Expected status 200 for %s", link.Href)
			}
		})
	})
}
