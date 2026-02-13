// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package listenerutil

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomHeadersWrapper(t *testing.T) {
	listenerConfig := &ListenerConfig{
		Type: "tcp",
		CustomApiResponseHeaders: map[int]http.Header{
			0: {
				"Test":                      {"default value", "default value 2"},
				"Content-Security-Policy":   {"default-src 'none'"},
				"X-Content-Type-Options":    {"nosniff"},
				"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
				"Cache-Control":             {"no-store"},
			},
			200: {
				"Test": {"200 value"},
			},
			2: {
				"Test": {"2xx value"},
			},
			401: {
				"Test": {"401 value"},
			},
			4: {
				"Test": {"4xx value"},
			},
		},
		CustomUiResponseHeaders: map[int]http.Header{
			0: {
				"Test":                      {"ui default value"},
				"Content-Security-Policy":   {"default-src 'none'; script-src 'self'; frame-src 'self'; font-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self'; media-src 'self'; manifest-src 'self'; style-src-attr 'self'; frame-ancestors 'self'"},
				"X-Content-Type-Options":    {"nosniff"},
				"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
				"Cache-Control":             {"max-age=604800"},
			},
			200: {
				"Test": {"ui 200 value"},
			},
			2: {
				"Test": {"ui 2xx value"},
			},
			401: {
				"Test": {"ui 401 value"},
			},
			4: {
				"Test": {"ui 4xx value"},
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
		expHeaders  map[string][]string
		expStatus   int
		wrapperFunc uiRequestFunc
	}{
		{
			name:   "200 api response",
			config: listenerConfig,
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Write([]byte("response"))
			}),
			expHeaders: map[string][]string{
				"Test":                      {"200 value"},
				"Content-Security-Policy":   {"default-src 'none'"},
				"X-Content-Type-Options":    {"nosniff"},
				"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
				"Cache-Control":             {"no-store"},
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
			expHeaders: map[string][]string{
				"Test":                      {"default value", "default value 2"},
				"Content-Security-Policy":   {"default-src 'none'"},
				"X-Content-Type-Options":    {"nosniff"},
				"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
				"Cache-Control":             {"no-store"},
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
			expHeaders: map[string][]string{
				"Test":                      {"4xx value"},
				"Content-Security-Policy":   {"default-src 'none'"},
				"X-Content-Type-Options":    {"nosniff"},
				"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
				"Cache-Control":             {"no-store"},
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
			expHeaders: map[string][]string{
				"Test":                      {"401 value"},
				"Content-Security-Policy":   {"default-src 'none'"},
				"X-Content-Type-Options":    {"nosniff"},
				"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
				"Cache-Control":             {"no-store"},
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
			expHeaders: map[string][]string{
				"Test":                      {"ui 200 value"},
				"Content-Security-Policy":   {"default-src 'none'; script-src 'self'; frame-src 'self'; font-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self'; media-src 'self'; manifest-src 'self'; style-src-attr 'self'; frame-ancestors 'self'"},
				"X-Content-Type-Options":    {"nosniff"},
				"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
				"Cache-Control":             {"max-age=604800"},
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
			expHeaders: map[string][]string{
				"Test":                      {"ui default value"},
				"Content-Security-Policy":   {"default-src 'none'; script-src 'self'; frame-src 'self'; font-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self'; media-src 'self'; manifest-src 'self'; style-src-attr 'self'; frame-ancestors 'self'"},
				"X-Content-Type-Options":    {"nosniff"},
				"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
				"Cache-Control":             {"max-age=604800"},
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
			expHeaders: map[string][]string{
				"Test":                      {"ui 4xx value"},
				"Content-Security-Policy":   {"default-src 'none'; script-src 'self'; frame-src 'self'; font-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self'; media-src 'self'; manifest-src 'self'; style-src-attr 'self'; frame-ancestors 'self'"},
				"X-Content-Type-Options":    {"nosniff"},
				"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
				"Cache-Control":             {"max-age=604800"},
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
			expHeaders: map[string][]string{
				"Test":                      {"ui 401 value"},
				"Content-Security-Policy":   {"default-src 'none'; script-src 'self'; frame-src 'self'; font-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self'; media-src 'self'; manifest-src 'self'; style-src-attr 'self'; frame-ancestors 'self'"},
				"X-Content-Type-Options":    {"nosniff"},
				"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
				"Cache-Control":             {"max-age=604800"},
			},
			expStatus:   401,
			wrapperFunc: uiRequest,
		},
		{
			name:   "empty response",
			config: listenerConfig,
			handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Add("Test2", "another value, ey")
			}),
			expHeaders: map[string][]string{
				"Test":                      {"200 value"},
				"Content-Security-Policy":   {"default-src 'none'"},
				"X-Content-Type-Options":    {"nosniff"},
				"Strict-Transport-Security": {"max-age=31536000; includeSubDomains"},
				"Cache-Control":             {"no-store"},
				"Test2":                     {"another value, ey"},
			},
			expStatus:   200,
			wrapperFunc: apiRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrappedHandler := WrapCustomHeadersHandler(tt.handler, tt.config, tt.wrapperFunc)

			r := httptest.NewRequest("GET", "http://localhost:9200/", nil)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, r)

			resp := w.Result()
			assert.Equal(t, tt.expStatus, resp.StatusCode)

			for header, expected := range tt.expHeaders {
				assert.Empty(t, cmp.Diff(expected, resp.Header.Values(header)))
			}
		})
	}
}

func TestWrappingPropagatesInterfaces(t *testing.T) {
	t.Parallel()

	type testNoOptional struct {
		http.ResponseWriter
	}
	type testPusherHijacker struct {
		http.ResponseWriter
		testHijacker
		testPusher
	}
	type testPusherFlusher struct {
		http.ResponseWriter
		testPusher
		testFlusher
	}
	type testFlusherHijacker struct {
		http.ResponseWriter
		testFlusher
		testHijacker
	}

	type testAll struct {
		http.ResponseWriter
		testFlusher
		testHijacker
		testPusher
	}

	tests := []struct {
		name         string
		wrap         http.ResponseWriter
		wantFlusher  bool
		wantPusher   bool
		wantHijacker bool
	}{
		{
			name: "missing-test-writer",
			wrap: &testFlusher{},
		},
		{
			name: "missing-wrapper",
		},
		{
			name: "success-no-optional",
			wrap: &testNoOptional{},
		},
		{
			name:        "success-flusher",
			wrap:        &testFlusher{},
			wantFlusher: true,
		},
		{
			name:        "success-flusher-hijacker",
			wrap:        &testFlusherHijacker{},
			wantFlusher: true,
		},
		{
			name:       "success-pusher",
			wrap:       &testPusher{},
			wantPusher: true,
		},
		{
			name:         "success-pusher-hijacker",
			wrap:         &testPusherHijacker{},
			wantHijacker: true,
			wantPusher:   true,
		},
		{
			name:        "success-pusher-flusher",
			wrap:        &testPusherFlusher{},
			wantFlusher: true,
			wantPusher:  true,
		},
		{
			name:         "success-hijacker",
			wrap:         &testHijacker{},
			wantHijacker: true,
		},
		{
			name:         "success-all",
			wrap:         &testAll{},
			wantHijacker: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert, require := assert.New(t), require.New(t)
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantPusher {
					_, ok := w.(http.Pusher)
					assert.Truef(ok, "wanted an response writer that satisfied the http.Pusher interface")
				}
				if tt.wantHijacker {
					_, ok := w.(http.Hijacker)
					assert.Truef(ok, "wanted an response writer that satisfied the http.Hijacker interface")
				}
				if tt.wantFlusher {
					f, ok := w.(http.Flusher)
					assert.Truef(ok, "wanted an response writer that satisfied the http.Flusher interface")
					f.Flush()
				}
			})
			wrapped := WrapCustomHeadersHandler(h, &ListenerConfig{}, func(r *http.Request) bool { return false })
			require.NotNil(wrapped)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			wrapped.ServeHTTP(rec, req)
			if tt.wantFlusher {
				assert.True(rec.Flushed)
			}
		})
	}

}

type testFlusher struct {
	http.ResponseWriter
	http.Flusher
}

func (t *testFlusher) Flush() {
	t.Flusher.Flush()
}

type testPusher struct {
	http.ResponseWriter
}

func (t *testPusher) Push(target string, opts *http.PushOptions) error { return nil }

type testHijacker struct {
	http.ResponseWriter
}

func (t *testHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) { return nil, nil, nil }
