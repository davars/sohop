package state

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSessionExpiresDeterministically(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		store, err := New("test", testSecret, "example.com")
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		assert.NoError(t, store.Authorize(rec, nil, "somebody"))
		cookie := rec.Result().Cookies()[0]
		assert.Equal(t, start.Add(sessionAge).Unix(), cookie.Expires.Unix())

		// GetSession caches per request, so build a fresh one per check.
		req := func() *http.Request {
			r := httptest.NewRequest("GET", "https://example.com/", nil)
			r.AddCookie(cookie)
			return r
		}
		assert.True(t, store.IsAuthorized(req()))

		time.Sleep(sessionAge - time.Second) // instant: fake clock
		assert.True(t, store.IsAuthorized(req()))

		time.Sleep(2 * time.Second)
		assert.False(t, store.IsAuthorized(req()))
	})
}
