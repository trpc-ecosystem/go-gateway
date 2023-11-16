# Mock Response Plugin

The mock response plugin allows you to return a fixed response without making a request to the backend service.

Use cases:
- Mocking interfaces for client debugging.
- Returning a fixed response when an interface is offline to prevent errors in older client versions.
- Providing a configuration management interface.

## Technical Solution

Intercept the request and return a pre-configured fixed response.

## Plugin Usage

### Import the Plugin in the Gateway Project's main.go

- Add the import statement

```go
import (
    _ "trpc.group/trpc-go/trpc-gateway/plugin/mocking"
)
```

- Configure the tRPC framework to enable the mocking interceptor.

Note: Make sure to register it in server.service.filter and not in server.filter.

```yaml
global:                             # Global configuration
server: # Server configuration
  filter:                                          # Interceptor list for all service handler functions
  service: # Business services provided, can have multiple
    - name: trpc.inews.trpc.gateway      # Route name of the service
      filter:
        - mocking # Gateway plugin registered in the service filter, so that it can be dynamically loaded in router.yaml
plugins: # Plugin configuration
  log:                                            # Log configuration
  gateway: # Plugin type is gateway
    mocking:  # Mock response plugin
```

#### Configure the Plugin in the Gateway's router.yaml File

Different levels of plugins are executed only once, with the priority order: router plugin > service plugin > global plugin.

```yaml
router:
  - method: /v1/user/info
    id: "xxxxxx"
    target_service:
      - service: trpc.user.service
    plugins:
      - name: mocking # Router-level plugin
        props:
          response_example: '{"code":0,"data":{}}' # Mock response body
          content_type: "" # Content-Type header of the response, default: application/json
          delay: 0 # Delay in milliseconds before returning the response, default: 0
          response_status: 200 # HTTP status code of the response, default: 200
          with_mock_header: true # When set to true, adds the response header "x-mock-by: tRPC-Gateway". When set to false, the header is not added.
          scale: true # Mock traffic ratio, in percentage. For example, if it is one in ten thousand, fill in: 0.01. The default is full-scale mock.
          hash_key: suid # Hash key for mocking traffic, providing the ability to perform grayscale testing based on request parameters.
client:
  - name: trpc.user.service
    plugins:
      - name: mocking # Service-level configuration
        props:
          response_example: '{"code":0,"data":{}}' # Mock response body
plugins:
  - name: mocking # Global configuration
    props:
      response_example: '{"code":0,"data":{}}' # Mock response body
```