// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package httputil

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type ignoreExtensionsRoundTripper struct {
	base         *http.Transport
	extsToIgnore []asn1.ObjectIdentifier
}

// Creates a RoundTripper that may be used in an HTTP client which will ignore the provided extensions if present
// on a certificate.
func NewIgnoreUnsupportedExtensionsRoundTripper(base http.RoundTripper, extsToIgnore []asn1.ObjectIdentifier) http.RoundTripper {
	if len(extsToIgnore) == 0 {
		return base
	}
	if base == nil {
		base = http.DefaultTransport
	}

	tp, ok := base.(*http.Transport)
	if !ok {
		// We don't know how to deal with this object, bail
		return base
	}

	return &ignoreExtensionsRoundTripper{base: tp, extsToIgnore: extsToIgnore}
}

func (i *ignoreExtensionsRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	var domain string
	if strings.ContainsRune(request.URL.Host, ':') {
		var err error
		domain, _, err = net.SplitHostPort(request.URL.Host)
		if err != nil {
			return nil, fmt.Errorf("could not parse domain from URL host %s", request.URL.Host)
		}
	} else {
		domain = request.URL.Host
	}

	// Only update our values if the end-user hasn't overridden anything we wanted to do.
	var perReqTransport *http.Transport
	if !i.base.TLSClientConfig.InsecureSkipVerify && i.base.TLSClientConfig.VerifyConnection == nil {
		perReqTransport = i.base.Clone()
		var tlsConfig *tls.Config
		if perReqTransport.TLSClientConfig == nil {
			tlsConfig = &tls.Config{
				ServerName: domain,
			}
		} else {
			tlsConfig = i.base.TLSClientConfig.Clone()
		}
		tlsConfig.ServerName = domain

		tlsConfig.InsecureSkipVerify = true
		connectionVerifier := customVerifyConnection(tlsConfig, i.extsToIgnore)
		tlsConfig.VerifyConnection = connectionVerifier

		perReqTransport.TLSClientConfig = tlsConfig
	} else {
		perReqTransport = i.base
	}
	return perReqTransport.RoundTrip(request)
}

func customVerifyConnection(tc *tls.Config, extToIgnore []asn1.ObjectIdentifier) func(tls.ConnectionState) error {
	return func(cs tls.ConnectionState) error {
		certs := cs.PeerCertificates

		serverName := cs.ServerName
		if cs.ServerName == "" {
			if tc.ServerName == "" {
				return fmt.Errorf("the ServerName in TLSClientConfig is required to be set when UnhandledExtensionsToIgnore has values")
			}
			serverName = tc.ServerName
		} else if cs.ServerName != tc.ServerName {
			return fmt.Errorf("x509: connection state server name (%s) does not match requested (%s)", cs.ServerName, tc.ServerName)
		}

		for _, cert := range certs {
			if len(cert.UnhandledCriticalExtensions) == 0 {
				continue
			}
			var remainingUnhandled []asn1.ObjectIdentifier
			for _, ext := range cert.UnhandledCriticalExtensions {
				shouldRemove := isExtInIgnore(ext, extToIgnore)
				if !shouldRemove {
					remainingUnhandled = append(remainingUnhandled, ext)
				}
			}
			cert.UnhandledCriticalExtensions = remainingUnhandled
		}

		opts := x509.VerifyOptions{
			Roots:         tc.RootCAs,
			DNSName:       serverName,
			Intermediates: x509.NewCertPool(),
		}

		for _, cert := range certs[1:] {
			opts.Intermediates.AddCert(cert)
		}

		_, err := certs[0].Verify(opts)
		if err != nil {
			return &tls.CertificateVerificationError{UnverifiedCertificates: certs, Err: err}
		}

		return nil
	}
}

func isExtInIgnore(ext asn1.ObjectIdentifier, ignoreList []asn1.ObjectIdentifier) bool {
	for _, extToIgnore := range ignoreList {
		if ext.Equal(extToIgnore) {
			return true
		}
	}

	return false
}
