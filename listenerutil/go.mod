module github.com/hashicorp/go-secure-stdlib/listenerutil

go 1.16

require (
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.1
	github.com/hashicorp/go-secure-stdlib/reloadutil v0.1.1
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.1
	github.com/hashicorp/go-secure-stdlib/tlsutil v0.1.1
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/hcl v1.0.0
	github.com/jefferai/isbadcipher v0.0.0-20190226160619-51d2077c035f
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mitchellh/cli v1.1.2
	github.com/stretchr/testify v1.7.0
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980 // indirect
)

replace (
	github.com/hashicorp/go-secure-stdlib/parseutil => ../parseutil
	github.com/hashicorp/go-secure-stdlib/reloadutil => ../reloadutil
	github.com/hashicorp/go-secure-stdlib/strutil => ../strutil
	github.com/hashicorp/go-secure-stdlib/tlsutil => ../tlsutil
)
