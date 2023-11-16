# Set Router

Set Router is used to set canary identifiers for specific requests.

## Use Cases

Specify the set node to forward requests to.
Note: There are limitations to the set mechanism itself. Enabling set routing will disable proximity-based routing, so careful evaluation is needed.

## Technical Solution

Uses the `WithCalleeSetName` options operation in tRPC-Go. Note that if the set node cannot be obtained, an empty node list will be returned, indicating a failed addressing.

## Plugin Usage

- Configure the client in the `router.yaml` file. To make cross-environment calls, set `disable_servicerouter` to `true`.
- Configure the `set_name` in the corresponding interface or client.

### Import the Plugin in the Gateway Project's `main.go` file

- Add the import statement

```go
import (
_ "trpc.group/trpc-go/trpc-gateway/plugin/polaris/setrouter"
)
```

- tRPC framework configuration file, enable the canaryrouter interceptor.

Note: Make sure to register it in server.service.filter, not server.filter.

```yaml
global:                             # Global configuration
server:                             # Server configuration
  filter:                          # Interceptor list for all service handlers
  service:                          # Business services provided, can have multiple
    - name: trpc.inews.trpc.gateway      # Service routing name
      filter:
        - setrouter: # Gateway plugin registered in the service filter, allowing dynamic loading in router.yaml
plugins:                            # Plugin configuration
  log:                                            # Log configuration
  gateway:                           # Plugin type is gateway
    setrouter:                       # Set Router plugin
```

#### Configure the Plugin in the Gateway's Router Configuration router.yaml file

Plugins at different levels will only be executed once, with the priority order: Router plugin > Service plugin > Global plugin

```yaml
router: # Router configuration
  - method: /v1/user/info
    id: "xxxxxx"
    target_service:
      - service: trpc.user.service
    plugins:
      - name: setrouter
        props:
          set_name: set.tj.1
client: # Upstream service configuration, follows the trpc protocol
  - name: trpc.user.service
    plugins:
      - name: setrouter
        props:
          set_name: set.tj.1
plugins:
```