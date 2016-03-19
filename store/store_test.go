package store

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const testSecret = "27e21c8d866594bd446c4a509d890ce2f59dcb26d89751b77ca236e5be3e0d7c26532a60e1ed9fd4f7b924e363d64e7a44a56dd57d84cf34eb7f0db0e19889f5"

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		domain string

		err string
	}{
		{
			err: ErrMissingName.Error(),
		},
		{
			name: "foo",
			err:  ErrMissingDomain.Error(),
		},
		{
			name:   "foo",
			domain: "foo",
			err:    "Session store secret should be a 128-character hex-encoded string.",
		},
		{
			name:   "foo",
			domain: "foo",
			secret: "27e21c8d866594bd446c4a509d890ce2f59dcb26d89751b77ca236e5be3e0d7c26532a60e1efd4f7b924e363d64e7a44a56dd57d84cf34eb7f0db0e19889f5",
			err:    "Session store secret should be a 128-character hex-encoded string.",
		},
		{
			name:   "foo",
			domain: "foo",
			secret: testSecret,
			err:    "<nil>",
		},
	}
	for _, test := range tests {
		_, err := New(test.name, test.secret, test.domain)
		if err == nil {
			err = fmt.Errorf("%v", nil)
		}
		if !strings.Contains(err.Error(), test.err) {
			t.Errorf("got %q, want something containing %q", err.Error(), test.err)
		}
	}
}

func TestStore(t *testing.T) {
	store, err := New("test", testSecret, "example.com")
	require.NoError(t, err)
	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	sess, err := store.Get(req, store.Name())
	require.NoError(t, err)
	sess.Values["foo"] = 42
	w := httptest.NewRecorder()
	sess.Save(req, w)

	cookieHeader := w.HeaderMap["Set-Cookie"]
	require.Equal(t, 1, len(cookieHeader))
	cookie := cookieHeader[0]
	for _, s := range []string{"test=", "Path=/; Domain=example.com; Expires=", "Max-Age=86400; HttpOnly; Secure"} {
		if !strings.Contains(cookie, s) {
			t.Fatalf("expected cookie to contain %q\nwas: %q", s, cookie)
		}
	}

	encoded := strings.Replace(strings.Split(cookie, ";")[0], "test=", "", 1)
	var v map[interface{}]interface{}
	err = store.(*namedStore).CookieStore.Codecs[0].Decode("test", encoded, &v)
	require.NoError(t, err)
	require.Equal(t, 42, v["foo"])

}
