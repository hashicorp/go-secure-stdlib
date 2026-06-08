module github.com/hashicorp/go-secure-stdlib/tlsutil

go 1.20

replace (
	github.com/hashicorp/go-secure-stdlib/parseutil => ../parseutil
	github.com/hashicorp/go-secure-stdlib/strutil => ../strutil
)

require (
	github.com/hashicorp/go-secure-stdlib/parseutil v0.2.0
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2
)

require (
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/hashicorp/go-sockaddr v1.0.6 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
)
