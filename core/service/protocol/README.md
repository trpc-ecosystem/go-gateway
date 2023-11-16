# Forward Protocol Support

This package is used to support other backend protocols and perform conversion operations from HTTP to various protocols
such as HTTP, tRPC, gRPC, and WebSocket.

### Directory Structure

- [cliprotocol.go](./cliprotocol.go) Protocol conversion interface definition
- [http](http) HTTP to HTTP conversion
- [trpc](trpc) HTTP to tRPC conversion
- [grpc](grpc) HTTP to tRPC conversion

You can implement the [cliprotocol.go](./cliprotocol.go) interface to enable custom protocol conversion for your
business.