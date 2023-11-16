# trpc error to response body plugin

In trpc HTTP interfaces, if an error is returned, the error information is included in the response headers, and the response body is empty. This is not user-friendly for client development.

This plugin provides a way to encapsulate the trpc error information from the response headers into the JSON response body, avoiding modifications to trpc interfaces.

Features：
- Specify the field name for the error code in the response body, supports multi-level settings, e.g., common.code, default: code. Supports specifying the data type, default: int.
- Specify the field name for the error message in the response body, supports multi-level settings, e.g., common.msg, default: msg.

## Technical Solution

- Extract the trpc-func-ret or trpc-ret from the response headers as the error code and set it in the JSON response body.
- Extract the trpc-error-msg as the error message and set it in the JSON response body.

## Plugin Usage

### Import the plugin in the main.go file of the gateway project

- Add import

```go
import (
  _ "trpc.group/trpc-go/trpc-gateway/plugin/transformer/trpcerr2body"
)
```

- Configure the trpcerr2body interceptor in the tRPC framework configuration file.

Note: Make sure to register it in server.service.filter, not in server.filter.

```yaml
global:                             # Global configuration
server:                             # Server configuration
  filter:                                          # Interceptor list for all service handlers
  service:                                         # Business services provided, can have multiple
    - name: trpc.inews.trpc.gateway                # Service routing name
      filter:
        - trpcerr2body                             # Gateway plugin registered in the service filter, so that it can be dynamically loaded in router.yaml
plugins:                            # Plugin configuration
  log:                                            # Log configuration
  gateway:                                         # Plugin type is gateway
    trpcerr2body:                                  # trpcerr2body response plugin
```

#### Configure the plugin in the router.yaml file of the gateway routing configuration

Different level plugins will only be executed once, with the priority: router plugin > service plugin > global plugin.

```yaml
router: # Router configuration
  - method: /v1/user/info
    id: "xxxxxx"
    target_service:
      - service: trpc.user.service
    plugins:
      - name: mocking # Router-level plugin:
        props:
          response_example: '{"code":0,"data":{}}' # Mock response body
          content_type: "" # Header Content-Type of the response, default: application/json
          delay: 0 # Delay time for the response in milliseconds, default: 0
          response_status: 200 # HTTP status code for the response, default: 200
          with_mock_header: true # When set to true, it adds the response header x-mock-by: tRPC-Gateway. When set to false, it does not add this response header.
client: # Upstream service configuration, consistent with the trpc protocol
  - name: trpc.user.service
    plugins:
      - name: trpcerr2body # Service-level configuration
        props:
          code_path: code # Error code field name, default: code, supports multi-level, e.g., common.code
          code_val_type: number # Code value type, number: int32, string: string, default: number
          msg_path: msg # Error message field name, default: msg, supports multi-level, e.g., common.msg
plugins:
  - name: trpcerr2body # Global configuration
    props:
```