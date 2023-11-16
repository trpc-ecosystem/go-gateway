# Gateway CORS Plugin

Developed based on the plugin at  https://git.woa.com/trpc-go/trpc-filter/tree/master/cors , it supports the ability to
dynamically load plugins in the gateway.

## Usage Instructions

### Import the Plugin in the main.go file of the Gateway Project

- Add the import statement

```go
import (
_ "trpc.group/trpc-go/trpc-gateway/plugin/cors"
)
```

- Configure the tRPC framework file to enable the CORS interceptor.

Note: Make sure to register it under server.service.filter, not server.filter.

```yaml
global:                             # Global configuration
server: # Server configuration
  filter:                                          # Interceptor list for all service handlers
  service: # Business services provided, can have multiple
    - name: trpc.inews.trpc.gateway      # Route name of the service
      filter:
        - cors # Gateway plugin registered as a filter in the service, so that it can be dynamically loaded in router.yaml
plugins: # Plugin configuration
  log:                                            # Log configuration
  gateway: # Plugin type is gateway
    cors:  # Name of the CORS plugin
```

Configure the plugin in the gateway routing configuration file router.yaml. It also supports global, service, and
router-level plugin configurations.

```yaml
router: # Routing configuration
  - method: /v1/user/info
    target_service:
      - service: trpc.user.service
    plugins:
      - name: cors # Router-level plugin: only effective for the current interface
        props:
          allow_origins: # Supported domains, supports suffix matching; if not specified, allows cross-origin requests from all domains. Corresponds to: Access-Control-Allow-Origin
            - xxx.qq.com
          allow_methods: # Supported HTTP methods, if not specified, supports all methods. Corresponds to: Access-Control-Request-Method
            - GET
            - POST
          allow_headers: # Allowed request headers, if not specified, supports all headers in preflight requests. Corresponds to: Access-Control-Allow-Headers
            - my-allow-header
          allow_credentials: true # Whether to allow credentials to be included. Corresponds to: Access-Control-Allow-Credentials
          expose_headers: # Exposed response headers. Corresponds to: Access-Control-Expose-Headers
            - my-expose-header
          max_age: 99999 # Preflight cache time, default is 0. Corresponds to: Access-Control-Max-Age
client: # Upstream service configuration, consistent with the trpc protocol
  - name: trpc.user.service
    plugins:
      - name: cors # Service-level configuration, effective for all interfaces forwarded to this service
        props:
plugins:
  - name: cors # Global configuration, effective for all interfaces
    props:
```

## References:

- CORS Specification — https://developer.mozilla.org/zh-CN/docs/Web/HTTP/CORS