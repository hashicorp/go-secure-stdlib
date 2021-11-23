package listenerutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/hashicorp/go-sockaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_WrapForwardedForHandler(t *testing.T) {
	t.Parallel()
	goodAddr, err := sockaddr.NewIPAddr("127.0.0.1")
	require.NoError(t, err)

	testErrResponseFn := func(w http.ResponseWriter, status int, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)

		type ErrorResponse struct {
			Errors []string `json:"errors"`
		}
		resp := &ErrorResponse{Errors: make([]string, 0, 1)}
		if err != nil {
			resp.Errors = append(resp.Errors, err.Error())
		}

		enc := json.NewEncoder(w)
		enc.Encode(resp)
	}

	badAddr, err := sockaddr.NewIPAddr("1.2.3.4")
	require.NoError(t, err)

	tests := []struct {
		name                   string
		listenerCfg            *ListenerConfig
		xForwardedFor          []string
		errRespFn              ErrResponseFn
		useNilHandler          bool
		wantFactoryErr         bool
		wantFactoryErrContains string
		wantErr                bool
		wantStatusCode         int
		wantBodyContains       string
	}{
		{
			name:                   "missing_listener_config",
			errRespFn:              testErrResponseFn,
			wantFactoryErr:         true,
			wantFactoryErrContains: "missing listener config: invalid parameter",
		},
		{
			name: "missing_err_resp_fn_config",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				return listenerConfig
			}(),
			wantFactoryErr:         true,
			wantFactoryErrContains: "missing response error function: invalid parameter",
		},
		{
			name: "missing_handler",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				return listenerConfig
			}(),
			useNilHandler:          true,
			errRespFn:              testErrResponseFn,
			wantFactoryErr:         true,
			wantFactoryErrContains: "missing http handler: invalid parameter",
		},
		{
			name: "reject_not_present_with_missing",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				return listenerConfig
			}(),
			errRespFn:        testErrResponseFn,
			wantStatusCode:   400,
			wantErr:          false,
			wantBodyContains: "missing x-forwarded-for",
		},
		{
			name: "reject_not_present_with_found",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				return listenerConfig
			}(),
			errRespFn:        testErrResponseFn,
			xForwardedFor:    []string{"1.2.3.4"},
			wantStatusCode:   200,
			wantErr:          false,
			wantBodyContains: "1.2.3.4:",
		},
		{
			name: "allow_unauth",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(badAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				return listenerConfig
			}(),
			errRespFn:        testErrResponseFn,
			xForwardedFor:    []string{"5.6.7.8"},
			wantStatusCode:   200,
			wantErr:          false,
			wantBodyContains: "127.0.0.1:",
		},
		{
			name: "fail_unauth",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(badAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				listenerConfig.XForwardedForRejectNotAuthorized = true
				return listenerConfig
			}(),
			errRespFn:        testErrResponseFn,
			xForwardedFor:    []string{"5.6.7.8"},
			wantStatusCode:   400,
			wantErr:          false,
			wantBodyContains: "not authorized for x-forwarded-for",
		},
		{
			name: "too_many_hops",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				listenerConfig.XForwardedForRejectNotAuthorized = true
				listenerConfig.XForwardedForHopSkips = 4
				return listenerConfig
			}(),
			errRespFn:        testErrResponseFn,
			xForwardedFor:    []string{"2.3.4.5,3.4.5.6"},
			wantStatusCode:   400,
			wantErr:          false,
			wantBodyContains: "would skip before earliest",
		},
		{
			name: "correct_hop_skipping",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				listenerConfig.XForwardedForRejectNotAuthorized = true
				listenerConfig.XForwardedForHopSkips = 1
				return listenerConfig
			}(),
			errRespFn:        testErrResponseFn,
			xForwardedFor:    []string{"2.3.4.5,3.4.5.6,4.5.6.7,5.6.7.8"},
			wantStatusCode:   200,
			wantErr:          false,
			wantBodyContains: "4.5.6.7:",
		},
		{
			name: "correct_hop_skipping_multi_header",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				listenerConfig.XForwardedForRejectNotAuthorized = true
				listenerConfig.XForwardedForHopSkips = 1
				return listenerConfig
			}(),
			errRespFn:        testErrResponseFn,
			xForwardedFor:    []string{"2.3.4.5", "3.4.5.6,4.5.6.7", "5.6.7.8"},
			wantStatusCode:   200,
			wantErr:          false,
			wantBodyContains: "4.5.6.7:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert, require := assert.New(t), require.New(t)
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(err)
			t.Cleanup(func() {
				_ = listener.Close()
			})
			testHandler, err := func() (http.Handler, error) {
				if tt.useNilHandler {
					return WrapForwardedForHandler(nil, tt.listenerCfg, tt.errRespFn)
				}
				h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(r.RemoteAddr))
				})
				return WrapForwardedForHandler(h, tt.listenerCfg, tt.errRespFn)
			}()
			if tt.wantFactoryErr {
				require.Error(err)
				if tt.wantFactoryErrContains != "" {
					assert.Containsf(err.Error(), tt.wantFactoryErrContains, "missing %s from err: %s", tt.wantFactoryErrContains, err.Error())
				}
				return
			}
			server := &http.Server{
				Handler: testHandler,
			}
			go func() {
				if err := server.Serve(listener); err != nil {
					if err != http.ErrServerClosed {
						require.NoError(err)
					}
				}
			}()
			t.Cleanup(func() {
				_ = server.Shutdown(context.Background())
			})

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/", listener.Addr().String()), nil)
			require.NoError(err)
			for _, h := range tt.xForwardedFor {
				req.Header.Add("x-forwarded-for", h)
			}
			resp, err := http.DefaultClient.Do(req)
			if tt.wantErr {
				return
			}
			require.NoError(err)
			require.Equalf(tt.wantStatusCode, resp.StatusCode, "expected error with %s status code and got: ", resp.StatusCode)
			require.NotNil(resp.Body)
			defer resp.Body.Close()
			var buf bytes.Buffer
			_, err = buf.ReadFrom(resp.Body)
			require.NoError(err)
			require.Containsf(buf.String(), tt.wantBodyContains, "bad error message: %v", err)
		})
	}
}

func Test_TrustedFromXForwardedFor(t *testing.T) {
	goodAddr, err := sockaddr.NewIPAddr("127.0.0.1")
	require.NoError(t, err)

	tests := []struct {
		name            string
		useNilReq       bool
		listenerCfg     *ListenerConfig
		xForwardedFor   []string
		remoteAddr      string
		wantAddr        *Addr
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "missing-req",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				return listenerConfig
			}(),
			useNilReq:       true,
			wantErr:         true,
			wantErrContains: "missing http request: invalid parameter",
		},
		{
			name:            "missing-listener-cfg",
			wantErr:         true,
			wantErrContains: "missing listener config: invalid parameter",
		},
		{
			name: "accept-not-present-with-missing",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = false
				return listenerConfig
			}(),
			wantErr: false,
		},
		{
			name: "accept-with-remote-addr-no-port",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = false
				return listenerConfig
			}(),
			xForwardedFor: []string{"127.0.0.1"},
			remoteAddr:    "localhost",
			wantErr:       false,
		},
		{
			name: "reject-with-remote-addr-no-port",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				return listenerConfig
			}(),
			xForwardedFor:   []string{"127.0.0.1"},
			remoteAddr:      "localhost",
			wantErr:         true,
			wantErrContains: "error parsing client hostport",
		},
		{
			name: "accept-with-invalid-remote-addr",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = false
				return listenerConfig
			}(),
			xForwardedFor: []string{"127.0.0.1"},
			remoteAddr:    "1.:22",
			wantErr:       false,
		},
		{
			name: "reject-with-invalid-remote-addr",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				return listenerConfig
			}(),
			xForwardedFor:   []string{"127.0.0.1"},
			remoteAddr:      "1.:22",
			wantErr:         true,
			wantErrContains: "error parsing client address: invalid IPAddr",
		},
		{
			name: "accept-with-invalid-x-forwaredfor-ip",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = false
				return listenerConfig
			}(),
			xForwardedFor: []string{"1.:22"},
			remoteAddr:    "127.0.0.1:22",
			wantErr:       false,
		},
		{
			name: "reject-with-invalid-x-forwaredfor-ip",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				return listenerConfig
			}(),
			xForwardedFor:   []string{"1.:22"},
			remoteAddr:      "127.0.0.1:22",
			wantErr:         true,
			wantErrContains: "error parsing client address",
		},
		{
			name: "accept-with-x-forwaredfor-including-port",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = false
				return listenerConfig
			}(),
			xForwardedFor: []string{"127.0.0.1:127"},
			remoteAddr:    "127.0.0.1:22",
			wantAddr:      &Addr{Host: "127.0.0.1", Port: "22"},
			wantErr:       false,
		},
		{
			name: "accept-with-x-forwaredfor-including-port-and-too-many-colons",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = false
				return listenerConfig
			}(),
			xForwardedFor: []string{"127.0.0.1::::127"},
			remoteAddr:    "127.0.0.1:22",
			wantErr:       false,
		},
		{
			name: "reject-with-x-forwaredfor-including-port-and-too-many-colons",
			listenerCfg: func() *ListenerConfig {
				listenerConfig := cfgListener(goodAddr)
				listenerConfig.XForwardedForRejectNotPresent = true
				return listenerConfig
			}(),
			xForwardedFor: []string{"127.0.0.1::::127"},
			remoteAddr:    "127.0.0.1:22",
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert, require := assert.New(t), require.New(t)
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(err)
			t.Cleanup(func() {
				_ = listener.Close()
			})
			var req *http.Request
			if !tt.useNilReq {
				req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/", listener.Addr().String()), nil)
				require.NoError(err)
				for _, h := range tt.xForwardedFor {
					req.Header.Add("x-forwarded-for", h)
				}
				if tt.remoteAddr != "" {
					req.RemoteAddr = tt.remoteAddr
				}
			}
			gotAddr, err := TrustedFromXForwardedFor(req, tt.listenerCfg)
			if tt.wantErr {
				require.Error(err)
				if tt.wantErrContains != "" {
					assert.Containsf(err.Error(), tt.wantErrContains, "missing %s from err: %s", tt.wantErrContains, err.Error())
				}
				return
			}
			require.NoError(err)
			if tt.wantAddr != nil {
				require.NotNil(gotAddr)
				assert.Equal(tt.wantAddr, gotAddr)
			} else {
				assert.Nil(gotAddr)
			}
		})
	}
}

func cfgListener(addr sockaddr.IPAddr) *ListenerConfig {
	return &ListenerConfig{
		XForwardedForAuthorizedAddrs: []*sockaddr.SockAddrMarshaler{
			{
				SockAddr: addr,
			},
		},
	}
}

func Test_OrigRemoteAddrFromCtx(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		ctx            context.Context
		wantRemoteAddr string
		wantNotOk      bool
	}{
		{
			name:      "missing-ctx",
			wantNotOk: true,
		},
		{
			name:      "missing-remote-addr",
			ctx:       context.Background(),
			wantNotOk: true,
		},
		{
			name: "valid",
			ctx: func() context.Context {
				ctx, err := newOrigRemoteAddrCtx(context.Background(), "host:port")
				require.NoError(t, err)
				return ctx
			}(),
			wantRemoteAddr: "host:port",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert, require := assert.New(t), require.New(t)
			got, ok := OrigRemoteAddrFromCtx(tt.ctx)
			if tt.wantNotOk {
				require.Falsef(ok, "should not have returned ok for the remote addr")
				assert.Emptyf(got, "should not have returned %q remote addr", got)
				return
			}
			require.Truef(ok, "should have been okay for getting a remote addr")
			require.NotEmpty(got, "remote addr should not be empty")
			assert.Equal(tt.wantRemoteAddr, got)
		})
	}

	factoryTests := []struct {
		name       string
		ctx        context.Context
		remoteAddr string
		wantErr    bool
	}{
		{"context", nil, "host:port", true},
		{"remote", context.Background(), "", true},
		{"valid", context.Background(), "host:port", false},
	}
	for _, tt := range factoryTests {
		t.Run(tt.name, func(t *testing.T) {
			assert, require := assert.New(t), require.New(t)
			ctx, err := newOrigRemoteAddrCtx(tt.ctx, tt.remoteAddr)
			if tt.wantErr {
				require.Error(err)
				assert.Nil(ctx)
				assert.ErrorIs(err, ErrInvalidParameter)
				return
			}
			require.NoError(err)
			assert.NotNil(ctx)
			got, found := OrigRemoteAddrFromCtx(ctx)
			assert.True(found)
			assert.Equal(tt.remoteAddr, got)
		})
	}

}
