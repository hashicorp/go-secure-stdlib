package listenerutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// type uiRequestFunc func(*http.Request) bool

func TestCustomHeadersWrapper(t *testing.T) {
	listenerConfig := &ListenerConfig{
		Type: "tcp",
		CustomApiResponseHeaders: map[int]map[string]string{
			0: {
				"Test":                      "default value; default value 2",
				"Content-Security-Policy":   "default-src 'none'",
				"X-Content-Type-Options":    "nosniff",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Cache-Control":             "no-store",
			},
			200: {
				"Test": "200 value",
			},
			2: {
				"Test": "2xx value",
			},
			401: {
				"Test": "401 value",
			},
			4: {
				"Test": "4xx value",
			},
		},
		CustomUiResponseHeaders: map[int]map[string]string{
			0: {
				"Test":                      "ui default value",
				"Content-Security-Policy":   "default-src 'none'; script-src 'self'; frame-src 'self'; font-src 'self'; connect-src 'self'; img-src 'self' data:*; style-src 'self'; media-src 'self'; manifest-src 'self'; style-src-attr 'self'; frame-ancestors 'self'",
				"X-Content-Type-Options":    "nosniff",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Cache-Control":             "max-age=604800",
			},
			200: {
				"Test": "ui 200 value",
			},
			2: {
				"Test": "ui 2xx value",
			},
			401: {
				"Test": "ui 401 value",
			},
			4: {
				"Test": "ui 4xx value",
			},
		},
	}

	uiRequest := func(*http.Request) bool {
		return true
	}
	apiRequest := func(*http.Request) bool {
		return false
	}

	tests := []struct {
		name        string
		config      *ListenerConfig
		handler     http.Handler
		expHeaders  map[string]string
		expStatus   int
		wrapperFunc uiRequestFunc
	}{
		{
			name:   "200 api response",
			config: listenerConfig,
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("response"))
			}),
			expHeaders: map[string]string{
				"Test":                      "200 value",
				"Content-Security-Policy":   "default-src 'none'",
				"X-Content-Type-Options":    "nosniff",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Cache-Control":             "no-store",
			},
			expStatus:   200,
			wrapperFunc: apiRequest,
		},
		{
			name:   "300 api response",
			config: listenerConfig,
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(300)
				w.Write([]byte("response"))
			}),
			expHeaders: map[string]string{
				"Test":                      "default value; default value 2",
				"Content-Security-Policy":   "default-src 'none'",
				"X-Content-Type-Options":    "nosniff",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Cache-Control":             "no-store",
			},
			expStatus:   300,
			wrapperFunc: apiRequest,
		},
		{
			name:   "400 api response",
			config: listenerConfig,
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(400)
				w.Write([]byte("response"))
			}),
			expHeaders: map[string]string{
				"Test":                      "4xx value",
				"Content-Security-Policy":   "default-src 'none'",
				"X-Content-Type-Options":    "nosniff",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Cache-Control":             "no-store",
			},
			expStatus:   400,
			wrapperFunc: apiRequest,
		},
		{
			name:   "401 api response",
			config: listenerConfig,
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(401)
				w.Write([]byte("response"))
			}),
			expHeaders: map[string]string{
				"Test":                      "401 value",
				"Content-Security-Policy":   "default-src 'none'",
				"X-Content-Type-Options":    "nosniff",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Cache-Control":             "no-store",
			},
			expStatus:   401,
			wrapperFunc: apiRequest,
		},
		// butts
		{
			name:   "200 ui response",
			config: listenerConfig,
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("response"))
			}),
			expHeaders: map[string]string{
				"Test":                      "ui 200 value",
				"Content-Security-Policy":   "default-src 'none'; script-src 'self'; frame-src 'self'; font-src 'self'; connect-src 'self'; img-src 'self' data:*; style-src 'self'; media-src 'self'; manifest-src 'self'; style-src-attr 'self'; frame-ancestors 'self'",
				"X-Content-Type-Options":    "nosniff",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Cache-Control":             "max-age=604800",
			},
			expStatus:   200,
			wrapperFunc: uiRequest,
		},
		{
			name:   "300 api response",
			config: listenerConfig,
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(300)
				w.Write([]byte("response"))
			}),
			expHeaders: map[string]string{
				"Test":                      "ui default value",
				"Content-Security-Policy":   "default-src 'none'; script-src 'self'; frame-src 'self'; font-src 'self'; connect-src 'self'; img-src 'self' data:*; style-src 'self'; media-src 'self'; manifest-src 'self'; style-src-attr 'self'; frame-ancestors 'self'",
				"X-Content-Type-Options":    "nosniff",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Cache-Control":             "max-age=604800",
			},
			expStatus:   300,
			wrapperFunc: uiRequest,
		},
		{
			name:   "400 api response",
			config: listenerConfig,
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(400)
				w.Write([]byte("response"))
			}),
			expHeaders: map[string]string{
				"Test":                      "ui 4xx value",
				"Content-Security-Policy":   "default-src 'none'; script-src 'self'; frame-src 'self'; font-src 'self'; connect-src 'self'; img-src 'self' data:*; style-src 'self'; media-src 'self'; manifest-src 'self'; style-src-attr 'self'; frame-ancestors 'self'",
				"X-Content-Type-Options":    "nosniff",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Cache-Control":             "max-age=604800",
			},
			expStatus:   400,
			wrapperFunc: uiRequest,
		},
		{
			name:   "401 api response",
			config: listenerConfig,
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(401)
				w.Write([]byte("response"))
			}),
			expHeaders: map[string]string{
				"Test":                      "ui 401 value",
				"Content-Security-Policy":   "default-src 'none'; script-src 'self'; frame-src 'self'; font-src 'self'; connect-src 'self'; img-src 'self' data:*; style-src 'self'; media-src 'self'; manifest-src 'self'; style-src-attr 'self'; frame-ancestors 'self'",
				"X-Content-Type-Options":    "nosniff",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
				"Cache-Control":             "max-age=604800",
			},
			expStatus:   401,
			wrapperFunc: uiRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wrappedHandler := WrapCustomHeadersHandler(tt.handler, tt.config, tt.wrapperFunc)

			r := httptest.NewRequest("GET", "http://localhost:9200/", nil)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, r)

			resp := w.Result()
			fmt.Printf("response: %#v", resp)
			assert.Equal(t, tt.expStatus, resp.StatusCode)

			for header, expected := range tt.expHeaders {
				assert.Equal(t, expected, resp.Header.Get(header))
			}
		})
	}
}
