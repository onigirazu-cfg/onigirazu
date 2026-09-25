package modules

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

func TestURI_RequestFromHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			w.Header().Set("X-Test", "yes")
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"ok":true}`)
		case "/echo":
			u, p, _ := r.BasicAuth()
			b, _ := io.ReadAll(r.Body)
			_, _ = io.WriteString(w, r.Method+" "+u+":"+p+" "+string(b))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	host := types.Host{Name: "localhost", Address: "127.0.0.1"}
	m := NewURIModule()
	ctx := context.Background()

	res, err := m.Execute(ctx, host, map[string]interface{}{"url": srv.URL + "/json"})
	require.NoError(t, err)
	require.True(t, res.Success, res.Error)
	assert.Equal(t, 200, res.Output["status"])
	assert.Equal(t, "yes", res.Output["headers"].(map[string]string)["X-Test"])
	assert.Equal(t, map[string]interface{}{"ok": true}, res.Output["json"])

	res, err = m.Execute(ctx, host, map[string]interface{}{
		"url": srv.URL + "/echo", "method": "POST", "body": "a b", "user": "u", "password": `p"w`,
	})
	require.NoError(t, err)
	require.True(t, res.Success, res.Error)
	assert.Equal(t, `POST u:p"w a b`, res.Output["text"])

	res, _ = m.Execute(ctx, host, map[string]interface{}{"url": srv.URL + "/missing", "status_code": 200})
	assert.False(t, res.Success, "404 must fail with status_code 200")
}
