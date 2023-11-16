# Fasthttp to net/http Forwarding

This conversion is implemented to proxy WebSocket and HTTP chunked protocols.

# Why Implement it Separately

The support for HTTP chunked protocol in the Fasthttp client is currently not complete. Additionally, forwarding
WebSocket and HTTP chunked protocols cannot fully utilize the advantages of Fasthttp. Therefore, the net/http package is
used as the client to implement the forwarding of WebSocket and HTTP chunked protocols.

# Implementation Reference

The implementation is based on the reverseproxy.go file in the net/http/httputil package.

# Usage Considerations

- Set the timeout in trpc_go.yaml server.service[0].timeout to 0 because WebSocket and HTTP chunked are long-lived connections.
- The timeout in router.yaml client[0].timeout will be forcibly changed to 0.
- HTTP chunked requests need to include the appropriate headers.
- For all body reading operations in the gateway, the following check should be performed to exclude stream requests:

```go
if fctx.IsBodyStream() {
    return "stream body"
}
```