// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package httputil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

var (
	inhibitAnyPolicyExt = asn1.ObjectIdentifier{2, 5, 29, 54}
	policyConstraintExt = asn1.ObjectIdentifier{2, 5, 29, 36}
)

func TestClient(t *testing.T) {
	for _, host := range []string{"localhost", "127.0.0.1", "[::1]"} {
		func() {
			srv := newTLSServer(t, true, host)
			defer srv.Close()
			runOverrideTests(host, t, srv)

			srv2 := newTLSServer(t, true, host)
			defer srv2.Close()

			url, err := url.Parse(srv2.URL)
			if err != nil {
				t.Fatalf("err parsing server address: %v", err)
			}

			runOverrideTests(url.Host, t, srv2)
		}()
	}
}

func runOverrideTests(host string, t *testing.T, srv *httptest.Server) {
	tests := []struct {
		name         string
		extsToIgnore []asn1.ObjectIdentifier
		errContains  string
	}{
		{
			name:        fmt.Sprintf("no-overrides-[%s]", host),
			errContains: "no extensions ignored",
		},
		{
			name:         fmt.Sprintf("partial-override-[%s]", host),
			extsToIgnore: []asn1.ObjectIdentifier{inhibitAnyPolicyExt},
			errContains:  "x509: unhandled critical extension",
		},
		{
			name:         fmt.Sprintf("full-override-[%s]", host),
			extsToIgnore: []asn1.ObjectIdentifier{inhibitAnyPolicyExt, policyConstraintExt},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			client, err := getClient(t, srv, tc.extsToIgnore)
			if err != nil {
				if tc.errContains == "" {
					t.Fatalf("unexpected error: %v", err)
				} else if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("expected error to contain '%s', got '%s'", tc.errContains, err.Error())
				} else {
					return
				}
			}
			resp, err := client.Get(srv.URL)
			if len(tc.errContains) > 0 {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("expected error to contain '%s', got '%s'", tc.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}

				defer func() { _ = resp.Body.Close() }()
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("got status code: %v", resp.StatusCode)
				}
			}
		})
	}
}

func getClient(t *testing.T, srv *httptest.Server, extsToIgnore []asn1.ObjectIdentifier) (*http.Client, error) {
	srvCertsRaw := srv.TLS.Certificates[0]
	rootCert, err := x509.ParseCertificate(srvCertsRaw.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed parsing root ca certificate: %v", err)
	}

	certpool := x509.NewCertPool()
	certpool.AddCert(rootCert)
	rt, err := NewIgnoreUnhandledExtensionsRoundTripper(&http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: certpool,
		},
	}, extsToIgnore)
	if err != nil {
		return nil, fmt.Errorf("error instantiating round tripper: %v", err)
	}

	client := http.Client{
		Transport: rt,
	}
	return &client, nil
}

func newTLSServer(t *testing.T, withUnsupportedExts bool, hostname string) *httptest.Server {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() { _ = req.Body.Close() }()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello World!"))
	}))

	// hack to force listening on IPv6
	if hostname[0] == '[' {
		if l, err := net.Listen("tcp6", "[::1]:0"); err != nil {
			panic(fmt.Sprintf("httptest: failed to listen on a port: %v", err))
		} else {
			ts.Listener = l
		}
	}

	ts.TLS = &tls.Config{Certificates: []tls.Certificate{getSelfSignedRoot(t, withUnsupportedExts)}}
	ts.StartTLS()
	ts.URL = strings.Replace(ts.URL, "127.0.0.1", hostname, 1)
	return ts
}

func getSelfSignedRoot(t *testing.T, withUnsupportedExts bool) tls.Certificate {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}
	pub := key.Public()

	inhibitExt := pkix.Extension{
		Id:       inhibitAnyPolicyExt,
		Critical: true,
		Value:    []byte{2, 1, 0},
	}

	polConstraint := pkix.Extension{
		Id:       policyConstraintExt,
		Critical: true,
		Value:    []byte{48, 6, 128, 1, 0, 129, 1, 0},
	}

	caTemplate := &x509.Certificate{
		Subject:      pkix.Name{CommonName: "Root CA with bad extensions"},
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-5 * time.Minute),
		NotAfter:     time.Now().Add(10 * time.Minute),
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	if withUnsupportedExts {
		caTemplate.ExtraExtensions = []pkix.Extension{polConstraint, inhibitExt}
		caTemplate.DNSNames = []string{"localhost"}
	} else {
		caTemplate.DNSNames = []string{"example.com"}
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, pub, key)
	if err != nil {
		t.Fatalf("failed to marshal CA certificate: %v", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{caBytes},
		PrivateKey:  key,
	}
}
