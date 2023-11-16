# Router Configuration Validation Plugin

Implement functionality similar to nginx -t, which checks the router configuration before updating it.

Usage:

- Configure a /gateway/check endpoint and add the routercheck plugin.
    - Note that this plugin will intercept requests and return a response, so it can only be configured under one endpoint.
    - Set an arbitrary upstream value.
- Send an HTTP request as shown in the example below.
    - It can be used with the Gateway Console to validate the configuration before publishing.

Security:

This plugin calls the router's CheckAndInit() method to validate and initialize the configuration. Consider the security implications of this operation:

- Will it affect the production configuration? CheckAndInit only validates and initializes the configuration, it does not update the production router configuration.
- Can unexpected external data cause a panic? Exception handling with defer has been added, so there is no risk.

## Technical Solution:

By calling the configured /gateway/check endpoint and passing the router configuration in YAML format, the plugin calls router.CheckAndInit() to validate the router configuration.

Example Request:

```http request
POST http://{gateway_host}/gateway/check
Content-Type: application/octet-stream

router: # Route configuration
  - method: ^/v1/user/ # Regex route
    is_regexp: true  # Whether it is a regex route, set to true to perform regex matching
    id: "path:^/v1/user/" # Route ID, used to identify a route for debugging (method will be duplicated)
    rewrite: /v1/user/info # Rewrite path
    target_service: # Upstream services
      - service: trpc.user.service # Service name, corresponding to the name in the client configuration
        weight: 10 # Service weight, the sum of weights cannot be 0
client: # Upstream service configuration, consistent with the trpc protocol
  - name: trpc.user.service
    namespace: Development
    network: tcp
    target: xxxx
    protocol: fasthttp
```

## Plugin Usage:

### Import the plugin in the main.go file of the gateway project

- Add the import statement:

```go
import (
_ "trpc.group/trpc-go/trpc-gateway/plugin/routercheck"
)
```

- tRPC framework configuration file, enable the routercheck interceptor.

Note: Make sure to register it in server.service.filter, not server.filter.

```yaml
global:                             # Global configuration
server: # Server configuration
  filter:                                          # Interceptor list for all service handlers
  service: # Business services provided, can have multiple
    - name: trpc.inews.trpc.gateway      # Service routing name
      filter:
        - routercheck # Gateway plugin registered in the service filter, allowing dynamic loading in router.yaml
plugins: # Plugin configuration
  log:                                            # Log configuration
  gateway: # Plugin type is gateway
    routercheck:  # Router configuration check
```

#### Configure the plugin in the gateway router configuration file (router.yaml)

```yaml
router: # 路由配置
  - method: /gateway/check
    id: "xxxxxx"
    target_service:
      - service: trpc.user.service
    plugins:
      - name: routercheck # 路由级别插件：腾讯网鉴权插件
client: # 上游服务配置，与trpc协议一致
  - name: trpc.user.service
    plugins:
plugins:
```