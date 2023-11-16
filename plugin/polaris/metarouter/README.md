# Polaris Metadata Routing Plugin

Forward requests to the target metadata service node based on the ArcticStar metadata in the request parameters.

## Use Cases

When user A applies to use service S, the backend service S receives the request and creates a separate container A,
allocating the corresponding resources. Container A registers its address information and label A with ArcticStar.
Subsequently, all requests from user A will be scheduled to container A and provided with the corresponding service.

Service S is generally a heavy-duty service that consumes a significant amount of resources, such as the popular AIGC
recently.

## Technical Solution

Parse the ArcticStar metadata in the request and set it in calleeMetadata.

## Plugin Usage

- Upstream client configuration needs to disable service routing by setting disable_servicerouter: true.
- It is recommended to add a prefix identifier to the metadata key to avoid key conflicts.
- For security reasons, if using user identification as metadata, you can parse the user identification through a
  business authentication plugin and set it in the request.

### Import the Plugin in the main.go file of the gateway project

- Add import statements

```go
import (
    _ "trpc.group/trpc-go/trpc-gateway/plugin/polaris/metarouter"
)
```

- Configure the tRPC framework file to enable the metarouter interceptor.

```yaml
global:                             # Global configuration
server:                             # Server configuration
  filter:                          # Interceptor list for all service handler functions
  service:                         # Business services provided, can have multiple
    - name: trpc.inews.trpc.gateway      # Route name for the service
      filter:
        - metarouter: # Gateway plugin registered in the service's filter, allowing dynamic loading in router.yaml
plugins:                            # Plugin configuration
  log:                              # Log configuration
  gateway:                          # Plugin type is gateway
    metarouter:                     # Metadata routing plugin
```

Note: Make sure to register it in server.service.filter, not server.filter.

#### Configure the Plugin in the gateway routing configuration router.yaml file

Plugins at different levels will only be executed once, with the priority order as follows: router plugin > service
plugin > global plugin

```yaml
router: # Router configuration
  - method: /v1/user/info
    id: "xxxxxx"
    target_service:
      - service: trpc.user.service
    plugins:
      - name: metarouter
        props:
          meta_key_list: # List of ArcticStar metadata keys
            - you-meta-key
client: # Upstream service configuration, consistent with the tRPC protocol
  - name: trpc.user.service
    plugins:
      - name: metarouter
        props:
          meta_key_list: # ArcticStar metadata key
            - you-meta-key
plugins:
```