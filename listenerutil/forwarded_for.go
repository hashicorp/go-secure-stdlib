package listenerutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/hashicorp/go-sockaddr"
)

type key int

const (
	remoteAddrKey key = iota

	missingPortErrStr = "missing port in address"
)

// ErrResponseFn provides a func to call whenever WrapForwardedForHandler
// encounters an error
type ErrResponseFn func(w http.ResponseWriter, status int, err error)

// WrapForwaredForHandler is an http middleware handler which uses the
// XForwardedFor* listener config settings to determine how/if X-Forwarded-For
// are trusted/allowed for an inbound request.  In the end, if a "trusted"
// X-Forwarded-For header is found, then the request RemoteAddr will be
// overwritten with it before the request is served.
func WrapForwardedForHandler(h http.Handler, l *ListenerConfig, respErrFn ErrResponseFn) (http.Handler, error) {
	if h == nil {
		return nil, fmt.Errorf("missing http handler: %w", ErrInvalidParameter)
	}
	if l == nil {
		return nil, fmt.Errorf("missing listener config: %w", ErrInvalidParameter)
	}
	if respErrFn == nil {
		return nil, fmt.Errorf("missing response error function: %w", ErrInvalidParameter)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		trusted, err := TrustedFromXForwardedFor(r, l)
		if err != nil {
			respErrFn(w, http.StatusBadRequest, err)
			return
		}
		if trusted == nil {
			h.ServeHTTP(w, r)
			return
		}
		newCtx, err := newOrigRemoteAddrCtx(r.Context(), r.RemoteAddr)
		if err != nil {
			respErrFn(w, http.StatusBadRequest, fmt.Errorf("error setting orig remote header ctx: %w", err))
			return
		}
		r = r.WithContext(newCtx)
		r.RemoteAddr = net.JoinHostPort(trusted.Host, trusted.Port)
		h.ServeHTTP(w, r)
		return
	}), nil
}

// Addr represents only the Host and Port of a TCP address.
type Addr struct {
	Host string
	Port string
}

// TrustedFromXForwardedFor will use the XForwardedFor* listener config settings
// to determine how/if X-Forwarded-For are trusted/allowed for an inbound
// request.  Important: return values of nil, nil are valid and simply means
// that no "trusted" header was found and no error was raised as well.  Errors
// can be raised for a number of conditions based on the listener config
// settings, especially when the config setting for
// XForwardedForRejectNotPresent is set to true which means if a "trusted"
// header can't be found the request should be rejected.
func TrustedFromXForwardedFor(r *http.Request, l *ListenerConfig) (*Addr, error) {
	if r == nil {
		return nil, fmt.Errorf("missing http request: %w", ErrInvalidParameter)
	}
	if l == nil {
		return nil, fmt.Errorf("missing listener config: %w", ErrInvalidParameter)
	}
	rejectNotPresent := l.XForwardedForRejectNotPresent
	hopSkips := l.XForwardedForHopSkips
	authorizedAddrs := l.XForwardedForAuthorizedAddrs
	rejectNotAuthz := l.XForwardedForRejectNotAuthorized

	headers, headersOK := r.Header[textproto.CanonicalMIMEHeaderKey("X-Forwarded-For")]
	if !headersOK || len(headers) == 0 {
		if !rejectNotPresent {
			return nil, nil
		}
		return nil, fmt.Errorf("missing x-forwarded-for header and configured to reject when not present")
	}

	// http request remote address will always have a host:port
	// (see: https://cs.opensource.google/go/go/+/refs/tags/go1.17.3:src/net/http/request.go;l=279-286)
	host, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If not rejecting treat it like we just don't have a valid
		// header because we can't do a comparison against an address we
		// can't understand
		if !rejectNotPresent {
			return nil, nil
		}
		return nil, fmt.Errorf("error parsing client hostport: %w", err)
	}

	addr, err := sockaddr.NewIPAddr(host)
	if err != nil {
		// We treat this the same as the case above
		if !rejectNotPresent {
			return nil, nil
		}
		return nil, fmt.Errorf("error parsing client address: %w", err)
	}

	var found bool
	for _, authz := range authorizedAddrs {
		if authz.Contains(addr) {
			found = true
			break
		}
	}
	if !found {
		// If we didn't find it and aren't configured to reject, simply
		// don't trust it
		if !rejectNotAuthz {
			return nil, nil
		}
		return nil, fmt.Errorf("client address not authorized for x-forwarded-for and configured to reject connection")
	}

	// At this point we have at least one value and it's authorized

	// Split comma separated ones, which are common. This brings it in line
	// to the multiple-header case.
	var acc []Addr
	for _, header := range headers {
		vals := strings.Split(header, ",")
		for _, v := range vals {
			// validate the header contains a valid IP
			v = strings.TrimSpace(v)
			h, p, err := net.SplitHostPort(v)
			switch {
			case err != nil && strings.Contains(err.Error(), missingPortErrStr):
				h = v
			case err != nil && !strings.Contains(err.Error(), missingPortErrStr):
				if !rejectNotPresent {
					return nil, nil
				}
				return nil, fmt.Errorf("error parsing client address host/port (%s) from header", v)
			}
			ip := net.ParseIP(h)
			if ip == nil {
				if !rejectNotPresent {
					return nil, nil
				}
				return nil, fmt.Errorf("error parsing client address (%s) from header", v)
			}
			acc = append(acc, Addr{Host: h, Port: p})
		}
	}

	indexToUse := int64(len(acc)) - 1 - hopSkips
	if indexToUse < 0 {
		// This is likely an error in either configuration or other
		// infrastructure. We could either deny the request, or we
		// could simply not trust the value. Denying the request is
		// "safer" since if this logic is configured at all there may
		// be an assumption it can always be trusted. Given that we can
		// deny accepting the request at all if it's not from an
		// authorized address, if we're at this point the address is
		// authorized (or we've turned off explicit rejection) and we
		// should assume that what comes in should be properly
		// formatted.
		return nil, fmt.Errorf("malformed x-forwarded-for configuration or request, hops to skip (%d) would skip before earliest chain link (chain length %d)", hopSkips, len(headers))
	}

	// TO-DO: using the remote address port here is not correct and still needs to be
	// resolved.  We could return the trusted port... but it could also be
	// empty.  Looking for some guidance from reviewers, especially on the
	// downstream vault affects if the port returned is empty
	return &Addr{acc[indexToUse].Host, port}, nil
}

// newOrigRemoteAddrCtx will return a context containing a value for the
// provided original remote address
func newOrigRemoteAddrCtx(ctx context.Context, origRemoteAddr string) (context.Context, error) {
	const op = "event.NewRequestInfoContext"
	if ctx == nil {
		return nil, fmt.Errorf("%s: missing context: %w", op, ErrInvalidParameter)
	}
	if origRemoteAddr == "" {
		return nil, fmt.Errorf("%s: missing original remote address: %w", op, ErrInvalidParameter)
	}
	return context.WithValue(ctx, remoteAddrKey, origRemoteAddr), nil
}

// OrigRemoteAddrFromCtx attempts to get the original remote address value from
// the context provided
func OrigRemoteAddrFromCtx(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	orig, ok := ctx.Value(remoteAddrKey).(string)
	return orig, ok
}
