// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package parseutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NormalizeAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		address  string
		expected string
		err      string
	}{
		{
			name:     "valid ipv4 address",
			address:  "127.0.0.1",
			expected: "127.0.0.1",
		},
		{
			name:     "valid ipv4 address with port",
			address:  "127.0.0.1:80",
			expected: "127.0.0.1:80",
		},
		{
			name:     "valid ipv4 address with port and path",
			address:  "127.0.0.1:80/test/path",
			expected: "127.0.0.1:80/test/path",
		},
		{
			name:     "valid ipv4 address with path",
			address:  "127.0.0.1/test/path",
			expected: "127.0.0.1/test/path",
		},
		{
			name:     "valid ipv4 uri with path",
			address:  "http://127.0.0.1/test/path",
			expected: "http://127.0.0.1/test/path",
		},
		{
			name:     "valid ipv4 uri with port and path",
			address:  "http://127.0.0.1:80/test/path",
			expected: "http://127.0.0.1:80/test/path",
		},
		{
			name:     "valid double colon address",
			address:  "::",
			expected: "::",
		},
		{
			name:     "valid ipv6 localhost address",
			address:  "::1",
			expected: "::1",
		},
		{
			name:     "valid ipv6 literal",
			address:  "2001:BEEF:0:0:0:1:0:0001",
			expected: "2001:beef::1:0:1",
		},
		{
			name:    "valid ipv6 literal with brackets",
			address: "[2001:BEEF:0:0:0:1:0:0001]",
			err:     "address cannot be encapsulated by brackets",
		},
		{
			name:     "valid ipv6 host:port",
			address:  "[2001:BEEF:0:0:0:1:0:0001]:80",
			expected: "[2001:beef::1:0:1]:80",
		},
		{
			name:     "valid ipv6 uri",
			address:  "https://[2001:BEEF:0:0:0:1:0:0001]",
			expected: "https://[2001:beef::1:0:1]",
		},
		{
			name:     "valid ipv6 uri with path",
			address:  "https://[2001:BEEF:0:0:0:1:0:0001]/test/path",
			expected: "https://[2001:beef::1:0:1]/test/path",
		},
		{
			name:     "valid ipv6 uri with port",
			address:  "https://[2001:BEEF:0:0:0:1:0:0001]:80",
			expected: "https://[2001:beef::1:0:1]:80",
		},
		{
			name:    "invalid ipv6 uri missing closing bracket",
			address: "https://[2001:BEEF:0:0:0:1:0:0001",
			err:     "failed to parse address",
		},
		{
			name:    "invalid ipv6 uri missing brackets",
			address: "https://2001:BEEF:0:0:0:1:0:0001",
			err:     "host contains an invalid IPv6 literal",
		},
		{
			name:    "invalid ipv6 literal",
			address: ":0:",
			err:     "failed to parse address",
		},
		{
			name:    "invalid ipv6 literal",
			address: "::0:",
			err:     "failed to parse address",
		},
		{
			name:    "invalid ipv6, not enough segments",
			address: "2001:BEEF:0:0:1:0:0001",
			err:     "host contains an invalid IPv6 literal",
		},
		{
			name:    "invalid ipv6 host:port, not enough segments",
			address: "[2001:BEEF:0:0:1:0:0001]:80",
			err:     "host contains an invalid IPv6 literal",
		},
		{
			name:    "invalid ipv6 literal with brackets, not enough segments",
			address: "[2001:BEEF:0:0:1:0:0001]",
			err:     "address cannot be encapsulated by brackets",
		},
		{
			name:    "invalid ipv6 uri, not enough segments",
			address: "https://[2001:BEEF:0:0:1:0:0001]:80",
			err:     "host contains an invalid IPv6 literal",
		},
		{
			name:    "invalid ipv6 uri withut port, not enough segments",
			address: "https://[2001:BEEF:0:0:1:0:0001]",
			err:     "host contains an invalid IPv6 literal",
		},
		{
			name:    "invalid ipv6, it's just brackets",
			address: "[]",
			err:     "address cannot be encapsulated by brackets",
		},
		{
			name:    "invalid address, empty",
			address: "",
			err:     "empty address",
		},
		{
			name:     "valid url with domain",
			address:  "https://www.google.com",
			expected: "https://www.google.com",
		},
		{
			name:     "valid url with domain and port",
			address:  "https://www.google.com:443",
			expected: "https://www.google.com:443",
		},
		{
			name:     "valid host with only sub domain",
			address:  "www.google.com",
			expected: "www.google.com",
		},
		{
			name:     "valid host:port with sub domain and port",
			address:  "www.google.com:443",
			expected: "www.google.com:443",
		},
		{
			name:     "valid host with only domain",
			address:  "google.com",
			expected: "google.com",
		},
		{
			name:     "valid host:port with domain and port",
			address:  "google.com:443",
			expected: "google.com:443",
		},
		{
			name:     "valid host with only dns name",
			address:  "hashicorp",
			expected: "hashicorp",
		},
		{
			name:     "valid host:port with dns name and port",
			address:  "hashicorp:443",
			expected: "hashicorp:443",
		},
		{
			name:    "invalid host with only dns name",
			address: "hashi corp",
			err:     "failed to parse address",
		},
		{
			name:     "valid url with path, schema, and subdomain",
			address:  "https://www.google.com/search?client=firefox-b-1-d&q=hey#section-1.2.3",
			expected: "https://www.google.com/search?client=firefox-b-1-d&q=hey#section-1.2.3",
		},
		{
			name:     "valid url with path but without schema or subdomain",
			address:  "google.com/search?client=firefox-b-1-d&q=hey#section-1.2.3",
			expected: "google.com/search?client=firefox-b-1-d&q=hey#section-1.2.3",
		},
		{
			name:     "valid uri with hostname and path",
			address:  "hashicorp/test/path?query=some&extra=data#section-1.2.3",
			expected: "hashicorp/test/path?query=some&extra=data#section-1.2.3",
		},
		{
			name:     "valid uri with crazy chars in query",
			address:  "hashicorp/test/path?I think actually anything can be past here !@#$^&*()[:]{;}",
			expected: "hashicorp/test/path?I think actually anything can be past here !@#$%5E&*()%5B:%5D%7B;%7D",
		},
		{
			name:     "valid uri with pre-encoded components",
			address:  "hashicorp/test/path?!@#$%5E&*()%5B:%5D%7B;%7D",
			expected: "hashicorp/test/path?!@#$%5E&*()%5B:%5D%7B;%7D",
		},
		{
			name: "valid uri with crazy chars in path",
			// note the lack of % as that would need to be encoded already
			address:  "hashicorp/test/path/ !@$^&*()[:]{;}",
			expected: "hashicorp/test/path/%20%21@$%5E&%2A%28%29%5B:%5D%7B;%7D",
		},
		{
			name:    "invalid uri with invalid percent encoding",
			address: "hashicorp/test/path?!@#$%^&*()[:]{;}",
			err:     "failed to parse address",
		},
		{
			name:    "invalid uri with invalid percent encoding",
			address: "hashicorp/test/path?%^&",
			// since there is nothing that needs to be encoded in this url,
			// url.Parse does not detect that `%^&` in an invalid url encoding
			expected: "hashicorp/test/path?%^&", // sad
		},
		{
			name:     "valid uri without schema, with ipv6",
			address:  "[2001:BEEF:0:0:0:1:0:0001]/test/path",
			expected: "[2001:beef::1:0:1]/test/path",
		},
		{
			name:     "valid host with user",
			address:  "dani@localhost",
			expected: "dani@localhost",
		},
		{
			name:     "valid uri no closing slash with frag",
			address:  "[2001:BEEF:0:0:0:1:0:0001]#test",
			expected: "[2001:beef::1:0:1]#test",
		},
		{
			name:     "valid uri with scheme, no closing slash, with frag",
			address:  "https://[2001:BEEF:0:0:0:1:0:0001]#test",
			expected: "https://[2001:beef::1:0:1]#test",
		},
		{
			name:     "valid ldap url with a bunch of data",
			address:  "ldap://ds.example.com:389/dc=example,dc=com?givenName,sn,cn?sub?(uid=john.doe)#extra",
			expected: "ldap://ds.example.com:389/dc=example,dc=com?givenName,sn,cn?sub?(uid=john.doe)#extra",
		},
		{
			name:     "valid ldap url with IPv6 address, port, and data",
			address:  "ldap://[2001:BEEF:0:0:0:1:0:0001]:389/dc=example,dc=com?givenName,sn,cn?sub?(uid=john.doe)#extra",
			expected: "ldap://[2001:beef::1:0:1]:389/dc=example,dc=com?givenName,sn,cn?sub?(uid=john.doe)#extra",
		},
		{
			name:     "valid ldap url with IPv6 address and data",
			address:  "ldap://[2001:BEEF:0:0:0:1:0:0001]/dc=example,dc=com?givenName,sn,cn?sub?(uid=john.doe)#extra",
			expected: "ldap://[2001:beef::1:0:1]/dc=example,dc=com?givenName,sn,cn?sub?(uid=john.doe)#extra",
		},
		{
			name:     "valid ldap url with IPv4 address, port, and data",
			address:  "ldap://127.0.0.1:389/dc=example,dc=com?givenName,sn,cn?sub?(uid=john.doe)#extra",
			expected: "ldap://127.0.0.1:389/dc=example,dc=com?givenName,sn,cn?sub?(uid=john.doe)#extra",
		},
		{
			name:     "valid ldap url with IPv4 address and data",
			address:  "ldap://127.0.0.1/dc=example,dc=com?givenName,sn,cn?sub?(uid=john.doe)#extra",
			expected: "ldap://127.0.0.1/dc=example,dc=com?givenName,sn,cn?sub?(uid=john.doe)#extra",
		},
		{
			name:     "valid ldap url with IPv6 address, no slash, and query data",
			address:  "ldap://[2001:BEEF:0:0:0:1:0:0001]:389?givenName,sn,cn?sub?(uid=john.doe)#extra",
			expected: "ldap://[2001:beef::1:0:1]:389?givenName,sn,cn?sub?(uid=john.doe)#extra",
		},
		{
			name:     "valid ldap url with IPv6 address, no slash, and frag data",
			address:  "ldap://[2001:BEEF:0:0:0:1:0:0001]:389#extra",
			expected: "ldap://[2001:beef::1:0:1]:389#extra",
		},
		{
			name:     "valid url with no scheme, ipv4 host, port address, and colon after port",
			address:  "127.0.0.1:80/test/path:123",
			expected: "127.0.0.1:80/test/path:123",
		},
		{
			name:     "valid url with no scheme, ipv6 host, port address, and colon after port",
			address:  "[2001:BEEF:0:0:0:1:0:0001]:80/test/path:123",
			expected: "[2001:beef::1:0:1]:80/test/path:123",
		},
		{
			name:    "anything other than numbers in port",
			address: "abc:gh",
			err:     "failed to parse address",
		},
		{
			name:    "invalid ipv4 host:port, host contains colon but no port",
			address: "127.0.0.1:",
			err:     "url has malformed host: missing port value after colon",
		},
		{
			name:    "invalid ipv6 host:port, host contains colon but no port",
			address: "[2001:4860:4860::8888]:",
			err:     "url has malformed host: missing port value after colon",
		},

		// imported from vault
		{
			name:     "hostname",
			address:  "vaultproject.io",
			expected: "vaultproject.io",
		},
		{
			name:     "hostname port",
			address:  "vaultproject.io:8200",
			expected: "vaultproject.io:8200",
		},
		{
			name:     "hostname URL",
			address:  "https://vaultproject.io",
			expected: "https://vaultproject.io",
		},
		{
			name:     "hostname port URL",
			address:  "https://vaultproject.io:8200",
			expected: "https://vaultproject.io:8200",
		},
		{
			name:     "hostname destination address",
			address:  "user@vaultproject.io",
			expected: "user@vaultproject.io",
		},
		{
			name:     "hostname destination address URL",
			address:  "http://user@vaultproject.io",
			expected: "http://user@vaultproject.io",
		},
		{
			name:     "hostname destination address URL port",
			address:  "http://user@vaultproject.io:8200",
			expected: "http://user@vaultproject.io:8200",
		},
		{
			name:     "ipv4",
			address:  "10.10.1.10",
			expected: "10.10.1.10",
		},
		{
			name:     "ipv4 IP:Port addr",
			address:  "10.10.1.10:8500",
			expected: "10.10.1.10:8500",
		},
		{
			name:     "ipv4 invalid IP:Port addr",
			address:  "[10.10.1.10]:8500",
			expected: "10.10.1.10:8500",
		},
		{
			name:     "ipv4 URL",
			address:  "https://10.10.1.10:8200",
			expected: "https://10.10.1.10:8200",
		},
		{
			name:     "ipv4 invalid URL",
			address:  "https://[10.10.1.10]:8200",
			expected: "https://10.10.1.10:8200",
		},
		{
			name:     "ipv4 destination address",
			address:  "username@10.10.1.10",
			expected: "username@10.10.1.10",
		},
		{
			name:     "ipv4 invalid destination address",
			address:  "username@10.10.1.10",
			expected: "username@10.10.1.10",
		},
		{
			name:     "ipv4 destination address port",
			address:  "username@10.10.1.10:8200",
			expected: "username@10.10.1.10:8200",
		},
		{
			name:     "ipv4 invalid destination address port",
			address:  "username@[10.10.1.10]:8200",
			expected: "username@10.10.1.10:8200",
		},
		{
			name:     "ipv4 destination address URL",
			address:  "https://username@10.10.1.10",
			expected: "https://username@10.10.1.10",
		},
		{
			name:     "ipv4 destination address URL port",
			address:  "https://username@10.10.1.10:8200",
			expected: "https://username@10.10.1.10:8200",
		},
		{
			name:     "ipv6 IP:Port RFC-5952 4.1 conformance leading zeroes",
			address:  "[2001:0db8::0001]:8500",
			expected: "[2001:db8::1]:8500",
		},
		{
			name:     "ipv6 RFC-5952 4.1 conformance leading zeroes",
			address:  "2001:0db8::0001",
			expected: "2001:db8::1",
		},
		{
			name:     "ipv6 URL RFC-5952 4.1 conformance leading zeroes",
			address:  "https://[2001:0db8::0001]:8200",
			expected: "https://[2001:db8::1]:8200",
		},
		{
			name:     "ipv6 bracketed destination address with port RFC-5952 4.1 conformance leading zeroes",
			address:  "username@[2001:0db8::0001]:8200",
			expected: "username@[2001:db8::1]:8200",
		},
		{
			name:     "ipv6 RFC-5952 4.2.2 conformance one 16-bit 0 field",
			address:  "2001:db8:0:1:1:1:1:1",
			expected: "2001:db8:0:1:1:1:1:1",
		},
		{
			name:     "ipv6 URL RFC-5952 4.2.2 conformance one 16-bit 0 field",
			address:  "https://[2001:db8:0:1:1:1:1:1]:8200",
			expected: "https://[2001:db8:0:1:1:1:1:1]:8200",
		},
		{
			name:     "ipv6 destination address with port RFC-5952 4.2.2 conformance one 16-bit 0 field",
			address:  "username@[2001:db8:0:1:1:1:1:1]:8200",
			expected: "username@[2001:db8:0:1:1:1:1:1]:8200",
		},
		{
			name:     "ipv6 RFC-5952 4.2.3 conformance longest run of 0 bits shortened",
			address:  "2001:0:0:1:0:0:0:1",
			expected: "2001:0:0:1::1",
		},
		{
			name:     "ipv6 URL RFC-5952 4.2.3 conformance longest run of 0 bits shortened",
			address:  "https://[2001:0:0:1:0:0:0:1]:8200",
			expected: "https://[2001:0:0:1::1]:8200",
		},
		{
			name:     "ipv6 destination address with port RFC-5952 4.2.3 conformance longest run of 0 bits shortened",
			address:  "username@[2001:0:0:1:0:0:0:1]:8200",
			expected: "username@[2001:0:0:1::1]:8200",
		},
		{
			name:     "ipv6 RFC-5952 4.2.3 conformance equal runs of 0 bits shortened",
			address:  "2001:db8:0:0:1:0:0:1",
			expected: "2001:db8::1:0:0:1",
		},
		{
			name:     "ipv6 URL no port RFC-5952 4.2.3 conformance equal runs of 0 bits shortened",
			address:  "https://[2001:db8:0:0:1:0:0:1]",
			expected: "https://[2001:db8::1:0:0:1]",
		},
		{
			name:     "ipv6 URL with port RFC-5952 4.2.3 conformance equal runs of 0 bits shortened",
			address:  "https://[2001:db8:0:0:1:0:0:1]:8200",
			expected: "https://[2001:db8::1:0:0:1]:8200",
		},
		{
			name:     "ipv6 destination address with port RFC-5952 4.2.3 conformance equal runs of 0 bits shortened",
			address:  "username@[2001:db8:0:0:1:0:0:1]:8200",
			expected: "username@[2001:db8::1:0:0:1]:8200",
		},
		{
			name:     "ipv6 RFC-5952 4.3 conformance downcase hex letters",
			address:  "2001:DB8:AC3:FE4::1",
			expected: "2001:db8:ac3:fe4::1",
		},
		{
			name:     "ipv6 URL RFC-5952 4.3 conformance downcase hex letters",
			address:  "https://[2001:DB8:AC3:FE4::1]:8200",
			expected: "https://[2001:db8:ac3:fe4::1]:8200",
		},
		{
			name:     "ipv6 destination address with port RFC-5952 4.3 conformance downcase hex letters",
			address:  "username@[2001:DB8:AC3:FE4::1]:8200",
			expected: "username@[2001:db8:ac3:fe4::1]:8200",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			actual, err := NormalizeAddr(tt.address)
			assert.Equal(tt.expected, actual)
			if tt.err != "" {
				require.Error(t, err)
				assert.ErrorContains(err, tt.err)
			} else {
				assert.Nil(err)
			}
		})
	}

}
