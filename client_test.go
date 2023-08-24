package testclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestHandler struct {
	Path       string
	Method     string
	Response   string
	StatusCode int
	Middleware func(t *testing.T, w http.ResponseWriter, r *http.Request)
}

func createTestServer(t *testing.T, handlers []TestHandler) http.Handler {
	mux := http.NewServeMux()

	for _, h := range handlers {
		handler := h // For closure
		mux.HandleFunc(handler.Path, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, handler.Method, r.Method)

			if handler.Middleware != nil {
				handler.Middleware(t, w, r)
			}

			w.WriteHeader(handler.StatusCode)
			_, _ = w.Write([]byte(handler.Response))
		})
	}

	return mux
}

func TestClient_New(t *testing.T) {
	server := http.NewServeMux()
	client := New(server)
	assert.NotNil(t, client)
}

func TestClient_Request(t *testing.T) {
	type want struct {
		code int
		body string
	}
	tests := []struct {
		name     string
		path     string
		method   string
		handlers []TestHandler
		want     want
	}{
		{
			name:   "When request is GET",
			path:   "/get",
			method: http.MethodGet,
			handlers: []TestHandler{
				{
					Path:       "/get",
					Method:     http.MethodGet,
					StatusCode: http.StatusOK,
					Response:   "ok",
				},
			},
			want: want{
				code: http.StatusOK,
				body: "ok",
			},
		},
		{
			name:   "When request is POST",
			path:   "/post",
			method: http.MethodPost,
			handlers: []TestHandler{
				{
					Path:       "/post",
					Method:     http.MethodPost,
					StatusCode: http.StatusOK,
					Response:   "ok",
				},
			},
			want: want{
				code: http.StatusOK,
				body: "ok",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer(t, tt.handlers)
			client := New(server)
			req := httptest.NewRequest(tt.method, tt.path, nil)
			client.Request(req)
			res := client.Response()

			assert.Equal(t, tt.want.code, res.StatusCode)

			body, _ := io.ReadAll(res.Body)
			assert.Equal(t, tt.want.body, string(body))
		})
	}
}

func TestClient_PostForm(t *testing.T) {
	type want struct {
		code int
		body string
	}
	tests := []struct {
		name     string
		path     string
		params   map[string]string
		handlers []TestHandler
		want     want
	}{
		{
			name: "When request has some parameters",
			path: "/post",
			params: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			handlers: []TestHandler{
				{
					Path:       "/post",
					Method:     http.MethodPost,
					StatusCode: http.StatusOK,
					Response:   "ok",
					Middleware: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
						contentType := r.Header.Get("Content-Type")
						assert.Equal(t, "application/x-www-form-urlencoded", contentType)

						err := r.ParseForm()
						assert.NoError(t, err)

						assert.Equal(t, "value1", r.PostFormValue("key1"))
						assert.Equal(t, "value2", r.PostFormValue("key2"))
					},
				},
			},
			want: want{
				code: http.StatusOK,
				body: "ok",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer(t, tt.handlers)
			client := New(server)
			client.PostForm(tt.path, tt.params)
			res := client.Response()

			assert.Equal(t, tt.want.code, res.StatusCode)

			body, _ := io.ReadAll(res.Body)
			assert.Equal(t, tt.want.body, string(body))
		})
	}
}

func TestClient_FollowRedirect(t *testing.T) {
	type want struct {
		code int
		body string
	}
	tests := []struct {
		name     string
		path     string
		params   map[string]string
		handlers []TestHandler
		want     want
		wantErr  bool
	}{
		{
			name: "When redirect with cookie",
			path: "/redirect",
			handlers: []TestHandler{
				{
					Path:       "/redirect",
					Method:     http.MethodGet,
					StatusCode: http.StatusFound,
					Middleware: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
						http.SetCookie(w, &http.Cookie{
							Name:  "cookie",
							Value: "candy",
						})
						http.Redirect(w, r, "/target", http.StatusFound)
					},
				},
				{
					Path:       "/target",
					Method:     http.MethodGet,
					Response:   "ok",
					StatusCode: http.StatusOK,
					Middleware: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
						cookie, err := r.Cookie("cookie")
						assert.NoError(t, err)
						assert.Equal(t, "candy", cookie.Value)
					},
				},
			},
			want: want{
				code: http.StatusOK,
				body: "ok",
			},
			wantErr: false,
		},
		{
			name: "When no redirect",
			path: "/no-redirect",
			handlers: []TestHandler{
				{
					Path:       "/no-redirect",
					Method:     http.MethodGet,
					Response:   "ok",
					StatusCode: http.StatusOK,
				},
			},
			want:    want{},
			wantErr: true,
		},
		{
			name: "When no location header",
			path: "/no-location",
			handlers: []TestHandler{
				{
					Path:       "/no-location",
					Method:     http.MethodGet,
					Response:   "ok",
					StatusCode: http.StatusFound,
				},
			},
			want:    want{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer(t, tt.handlers)
			client := New(server)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			client.Request(req)

			err := client.FollowRedirect()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			res := client.Response()
			assert.Equal(t, tt.want.code, res.StatusCode)

			body, _ := io.ReadAll(res.Body)
			assert.Equal(t, tt.want.body, string(body))
		})
	}
}
