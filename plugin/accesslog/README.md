# Access Log Plugin

You can use this plugin to record request logs to a local file or report them to a remote log platform.

## Usage

- Use the configuration method of the log module in the trpc-go framework.
- If the `accesslog` logger is not configured, the default trpc logger will be used for output.
- You can print your own business fields by configuring the plugin or overriding the DefaultBusinessFields method.

### Import the Plugin in the main.go of the Gateway Project

- Add the import statement

```go
import (
_ "trpc.group/trpc-go/trpc-gateway/plugin/accesslog"
)
```

- trpc_go.yaml framework configuration file, enable the accesslog interceptor.

Note: Make sure to register it in server.service.filter, not in server.filter.

```yaml
global:                             # Global configuration
server:                             # Server configuration
  filter:                          # Interceptor list for all service handler functions
  service:                         # Business services provided, can have multiple
    - name: trpc.inews.trpc.gateway      # Route name of the service
      filter:
        - accesslog # Gateway plugin registered in the service filter, so that it can be dynamically loaded in router.yaml
plugins:                            # Plugin configuration
  gateway:                          # Plugin type is gateway
    accesslog:                      # Access log plugin
  log:                              # Log configuration
    accesslog:                      # Configure access log output
      - writer: console             # Console standard output (default)
        level: debug                # Log level for standard output
      - writer: file                # Local file log
        level: debug                # Log level for local file rolling log
        writer_config:               # Specific configuration for local file output
          log_path: ${log_path}      # Local file log path
          filename: access.log       # Local file log filename
          roll_type: size            # File rolling type, size for rolling by size
          max_age: 7                 # Maximum number of days to keep logs
          max_size: 10               # Size of local file rolling log in MB
          max_backups: 10            # Maximum number of log files
          compress: false            # Whether to compress log files
      - writer: atta                # Other remote logs, such as Tencent's internal atta log platform
        level: info
        remote_config:
          agent_address: ${atta_agent_address}
          atta_id: 'xxx'
          atta_token: 'xxx'
          message_key: msg
          field:                     # Note: the order of fields reported to atta cannot be changed
            - path
            - upstream_path
            - router_id
            - err_no
            - err_msg
            - local_ip
            - upstream_service
            - upstream_protocol
            - upstream_addr
            - upstream_status
            - upstream_response_time
            - remote_addr
            - traceid
            - user_agent
            - host
            - referer
            - server_protocol
            - xxx  # Other business fields
```

Configure the plugin in the router.yaml file of the gateway, supporting global, service, and router-level plugin configurations.

```yaml
router: # Router configuration
  - method: /v1/user/info
    target_service:
      - service: trpc.user.service
    plugins:
      - name: accesslog
        props:
          field_list:
            - log_key: request_key # Business field
client: # Upstream service configuration, consistent with the trpc protocol
  - name: trpc.user.service
    plugins:
      - name: accesslog # Service-level configuration, effective for interfaces forwarded to this service
        props:
plugins:
  - name: accesslog # Global configuration, effective for all interfaces
    props:
```
