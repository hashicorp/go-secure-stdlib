// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configutil

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// general delimiters as defined in RFC 3986
// See: https://www.rfc-editor.org/rfc/rfc3986#section-2.2
const genDelims = ":/?#[]@"

func normalizeHostPort(host string, port string, url bool) (string, error) {
	// fmt.Println("host:", host, "port:", port)
	if host == "" {
		return "", nil
	}
	if ip := net.ParseIP(host); ip != nil {
		if url && ip.To4() == nil && port == "" {
			// this is a unique case, host is ipv6 and requires brackets due to
			// being part of a url, but they won't be added by net.JoinHostPort
			// as there is no port
			return "[" + ip.String() + "]", nil
		}
		host = ip.String()
	} else if strings.Contains(host, ":") {
		// host is an invalid ipv6 literal.
		// hosts cannot contain certain reserved characters, including ":"
		// See: https://www.rfc-editor.org/rfc/rfc3986#section-3.2.2,
		//      https://www.rfc-editor.org/rfc/rfc3986#section-2.2
		return "", fmt.Errorf("host contains an invalid IPv6 literal")
	}
	if port == "" {
		return host, nil
	}
	return net.JoinHostPort(host, port), nil
}

// NormalizeAddr takes an address as a string and returns a normalized copy.
// If the addr is a URL, IP Address, or host:port address that includes an IPv6
// address, the normalized copy will be conformant with RFC-5942 §4
//
// Valid formats include:
//   - host
//   - host:port
//   - scheme://user@host/path?query#frag
//
// Note: URLs and URIs must conform with https://www.rfc-editor.org/rfc/rfc3986#section-3
// or else the returned address may have been parsed and formatted incorrectly
//
// See: https://rfc-editor.org/rfc/rfc5952.html
func NormalizeAddr(address string) (string, error) {
	if address == "" {
		return "", fmt.Errorf("empty or invalid hostname")
	}

	if ip := net.ParseIP(address); ip != nil {
		return ip.String(), nil
	}

	if strings.HasPrefix(address, "[") && strings.HasSuffix(address, "]") {
		return NormalizeAddr(address[1 : len(address)-1])
	}

	if host, port, err := net.SplitHostPort(address); err == nil {
		return normalizeHostPort(host, port, false)
	}

	if u, err := url.ParseRequestURI(address); err == nil {
		if u.Host, err = normalizeHostPort(u.Hostname(), u.Port(), true); err != nil {
			return "", err
		}
		return u.String(), nil
	}
	// if the provided address does not have a scheme provided, attempt to
	// provide one and re-parse the result. this is done by looking for the
	// first general delimiter and checking if it exists or if it's not a colon
	// See: https://www.rfc-editor.org/rfc/rfc3986#section-3
	if idx := strings.IndexAny(address, genDelims); idx < 0 || address[idx] != ':' {
		const scheme = "https://"
		if u, err := url.ParseRequestURI(scheme + address); err == nil {
			if u.Host, err = normalizeHostPort(u.Hostname(), u.Port(), true); err != nil {
				return "", err
			}
			return strings.TrimPrefix(u.String(), scheme), nil
		}
	}

	return "", fmt.Errorf("unable to normalize given address")
}
