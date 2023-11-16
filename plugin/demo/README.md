# tRPC-Gateway Gateway Demo Plugin

This is a gateway plugin example that can be used as a template to quickly develop your own gateway plugins for your business.

Note that this documentation currently only applies to the development of plugins for the HTTP protocol, which is the main supported protocol of tRPC-Gateway.

Prerequisite Knowledge:
- [fasthttp API](https://github.com/valyala/fasthttp)

## Usage Instructions

### Import the Plugin in the main.go file of the Gateway Project

- Add the import statement

```go
import (
    _ "trpc.group/trpc-go/trpc-gateway/plugin/demo"
)
```

- trpc_go.yaml framework configuration file, enable the demo interceptor.

Note: Make sure to register it under server.service.filter, not server.filter.

```yaml
global:                             # Global configuration
server:                             # Server configuration
  filter:                          # Interceptor list for all service handler functions
  service:                          # Business services provided, can have multiple
    - name: trpc.inews.trpc.gateway      # Route name of the service
      filter:
        - demo                      # Gateway plugin registered as a filter in the service, so that it can be dynamically loaded in router.yaml
plugins:                            # Plugin configuration
  log:                              # Log configuration
  gateway:                          # Plugin type is gateway
    demo:                           # Name of the CORS plugin
```

Configure the plugin in the gateway routing configuration router.yaml, supporting global, service, and router-level plugin configurations.

```yaml
router: # Routing configuration
  - method: /v1/user/info
    target_service:
      - service: trpc.user.service
    plugins:
      - name: demo
        props:
          suid_name: xxx
client: # Upstream service configuration, follows the tRPC protocol
  - name: trpc.user.service
    plugins:
      - name: request_transformer  # Service-level configuration, will be effective for all interfaces forwarded to this service
        props:
plugins:
  - name: demo                      # Global configuration, will be effective for all interfaces
    props:
```