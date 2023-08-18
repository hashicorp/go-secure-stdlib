# Counter Example

This example is based on `examples/bidirectional` from the go-plugin repository.

It uses a plugin to increment named counters, and the plugin uses a gRPC storage
service provided to it by the client. This is pretty similar to how storage works
in Vault plugins, where Vault exposes an isolated storage view for any durable
data that plugins need to persist.

To run the example:

```sh
# Build the plugin container image
$ go build -o go-plugin-counter ./plugin-counter && docker build -t go-plugin-counter .

# Read and write
$ go run main.go increment hello 1
$ go run main.go increment hello 2
$ go run main.go increment world 2
$ go run main.go get hello
```

## Updating the Protocol

If you update the .proto files, you can regenerate the .pb.go files using
[buf](https://github.com/bufbuild/buf):

```sh
$ buf generate
```
