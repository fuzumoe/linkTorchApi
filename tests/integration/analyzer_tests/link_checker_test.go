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

func TestLinkChecker_Integration(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusOK)
			} else if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			}
		case "/get":
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("GET OK"))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	_, err := url.Parse(ts.URL)
	require.NoError(t, err)

	lc := analyzer.NewLinkChecker(2, 5*time.Second)
	require.NotNil(t, lc)

	lcValue := reflect.ValueOf(lc).Elem()
	clientField := lcValue.FieldByName("client")
	require.True(t, clientField.IsValid(), "client field must exist")
	ptrToClient := unsafe.Pointer(clientField.UnsafeAddr())
	reflect.NewAt(clientField.Type(), ptrToClient).Elem().Set(reflect.ValueOf(ts.Client()))

	link1 := model.Link{Href: ts.URL + "/ok"}
	link2 := model.Link{Href: ts.URL + "/get"}
	links := []model.Link{link1, link2}

	t.Run("Integration: Run LinkChecker", func(t *testing.T) {
		updatedLinks := lc.Run(context.Background(), links)
		require.Len(t, updatedLinks, 2, "Expected two links to be returned")

		t.Run("Verify Link Statuses", func(t *testing.T) {
			for _, link := range updatedLinks {
				require.Equal(t, http.StatusOK, link.StatusCode, "Expected status 200 for %s", link.Href)
			}
		})
	})
}
