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

	"github.com/hashicorp/go-hclog"
)

type ignoreExtensionsRoundTripper struct {
	base         *http.Transport
	extsToIgnore []asn1.ObjectIdentifier
	logger       hclog.Logger
}

// NewIgnoreUnhandledExtensionsRoundTripper creates a RoundTripper that may be used in an HTTP client which will
// ignore the provided extensions if presently unhandled on a certificate.  If base is nil, the default RoundTripper is used.
func NewIgnoreUnhandledExtensionsRoundTripper(logger hclog.Logger, base http.RoundTripper, extsToIgnore []asn1.ObjectIdentifier) http.RoundTripper {
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

	return &ignoreExtensionsRoundTripper{base: tp, logger: logger, extsToIgnore: extsToIgnore}
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
		connectionVerifier := i.customVerifyConnection(tlsConfig)
		tlsConfig.VerifyConnection = connectionVerifier

		perReqTransport.TLSClientConfig = tlsConfig
	} else {
		perReqTransport = i.base
	}
	return perReqTransport.RoundTrip(request)
}

func (i *ignoreExtensionsRoundTripper) customVerifyConnection(tc *tls.Config) func(tls.ConnectionState) error {
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
				shouldRemove := i.isExtInIgnore(ext)
				if shouldRemove {
					if i.logger != nil && i.logger.IsDebug() {
						i.logger.Debug("x509: ignoring unhandled extension", "oid", ext.String())
					}
				} else {
					remainingUnhandled = append(remainingUnhandled, ext)
				}
			}
			cert.UnhandledCriticalExtensions = remainingUnhandled
			if len(remainingUnhandled) > 0 && i.logger != nil {
				for _, ext := range remainingUnhandled {
					i.logger.Warn("x509: unhandled critical extension", "oid", ext.String())
				}
			}
		}

		// Now verify with the requested extensions removed
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

func (i *ignoreExtensionsRoundTripper) isExtInIgnore(ext asn1.ObjectIdentifier) bool {
	for _, extToIgnore := range i.extsToIgnore {
		if ext.Equal(extToIgnore) {
			return true
		}
	}

	return false
}
