# Router configuration etcd loader

## Usage:

- Create a key named "router_conf" in etcd to manage the router configuration.
- Import the etcd loader anonymously in the main.go file of the project.

```go
import    _ "trpc.group/trpc-go/trpc-gateway/core/loader/etcd"
```

- Specify the etcd loader in the trpc_go.yaml framework configuration and configure the etcd address.

```yaml
global: # Global configuration
  conf_provider: etcd        # Specify the etcd loader
server:
  app: ${app}                                               # Application name
  server: ${server}                                         # Process server name
  service: # Services provided by the business
    - name: trpc.${app}.${server}.gateway    # Service route name, replace ReplaceMe with your own service name, do not change app and server placeholders
plugins:
  # Configure etcd information, refer to: https://git.woa.com/trpc-go/trpc-config-etcd
  config:
    etcd:
      endpoints: ["http://127.0.0.1:2380"]
```

- Configuration example：[etcd loader example](../../../example/loader/etcd)
