# Canary Router

Set canary flag for specified requests.

## Use Cases

Set canary flag for specified requests and forward them to canary nodes in the upstream, in order to validate new features and identify issues in the gray stage.

## Technical Solution

Retrieve the parameter value based on the key set in the gateway plugin. If the value is in the configured value list, set the canary flag.

## Plugin Usage

- Enable Polaris canary routing in the gateway and upstream trpc framework configuration.
- Configure the canaryrouter plugin in the routing item with canary_key and values
- Set canary flag for upstream nodes
  - Set the canary:1 instance label for nodes in the Polaris console
- Note: Currently, Polaris canary routing only takes effect in the production environment

### Import the Plugin in the main.go of the Gateway Project

```go
import (
   _ "trpc.group/trpc-go/trpc-gateway/plugin/polaris/canaryrouter"
)
```

- Configure the canaryrouter interceptor in the tRPC framework configuration file.

Note: Make sure to register it in server.service.filter, not in server.filter

```yaml
global:                             # Global configuration
server: # Server configuration
  filter:                                          # Interceptor list for all service handler functions
  service: # Business services provided, can have multiple
    - name: trpc.inews.trpc.gateway      # Routing name of the service
      filter:
        - canaryrouter: # Gateway plugin registered in the service filter, so that it can be dynamically loaded in router.yaml
plugins: # Plugin configuration
  log:                                            # Log configuration
  gateway: # Plugin type is gateway
    canaryrouter:  # Canary router plugin
```

#### Configure the Plugin in the Gateway Routing Configuration router.yaml

Different level plugins are executed only once, with the priority: routing plugin > service plugin > global plugin

```yaml
router: # Routing configuration
  - method: /v1/user/info
    id: "xxxxxx"
    target_service:
      - service: trpc.user.service
    plugins:
      - name: canaryrouter
        props:
          request_key: user_id # Key of the request parameter to set the canary flag, if set_all is set, the canary flag will be set for all requests, and the parameters in query, header, and Cookie will be queried in order
          values: [ "xxx" ] # Value list of the request parameter to set the canary flag
          scale: 1 # Canary traffic ratio, in percentage, e.g., 0.01 for one in ten thousand
          hash_key: qimei36 # Canary traffic hash key
          canary_tag_val: my_canary # Polaris canary tag value, used with 123 to customize canary tags, default is 1
client: # Upstream service configuration, consistent with the trpc protocol
  - name: trpc.user.service
    plugins:
      - name: canaryrouter
        props:
plugins:
```