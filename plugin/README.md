# tRPC-Gateway Plugin

- [Background](#Background)
- [RPC-Gateway Plugin Development](#插件开发)
- [Phases of Plugin Execution](#插件执行的请求阶段)

## Background

Similar to other gateways in the industry, tRPC-Gateway provides the ability to extend the gateway through plugins.

## 插件开发

tRPC-Gateway plugins are essentially tRPC-Go plugins, so developers who are familiar with tRPC-Go plugin development
will find it extremely convenient to develop tRPC-Gateway plugins.

However, there are differences in functionality between the two.

|        Name         |               Scope                |      Registration Method      | Configuration Validation Timing	                | Plugin Configuration Retrieval Timing           |
|:-------------------:|:----------------------------------:|:-----------------------------:|-------------------------------------------------|-------------------------------------------------|
|   tRPC-Go Plugin    |      Applies to all requests       | Registered on service startup | Configuration validated on service startup      | Configuration retrieved on service startup      |
| tRPC-Gateway Plugin | Applies only to specified requests |    Dynamically registered     | Configuration validated on dynamic registration | Configuration retrieved during plugin execution |

Based on the above differences, tRPC-Gateway customizes the following rules in plugin development and registration,
based on the tRPC-Go plugin development：

- Gateway Plugin Development
    - Implement the [GatewayPlugin](plugin.go) interface。The CheckConfig method is used to validate the gateway
      configuration during dynamic registration.
    - Use `gwmsg.GwMessage(ctx).PluginConfig({pluginName})` in the ServerFilter method to retrieve the plugin
      configuration, as shown in the [demo plugin](demo/demo.go),corresponding to the configuration of plugins[0].prop
      in router.yaml.
- Gateway Plugin Registration
    - Import the corresponding plugin in main.go, similar to tRPC-Go plugins.
    - Register the gateway plugin in trpc_go.yaml under `server.service[0].filter`. Use `server.filter` to register
      tRPC-Go plugins.
- Gateway Plugin Execution
    - The plugin will only take effect for requests that have the gateway plugin configured in router.yaml. For detailed
      configuration, refer to the [Routing Configuration](../core/router/README.md) section, specifically the plugins
      chapter.
- Other Extension Capabilities
    - Custom error code and HTTP code mappings can be added using the [Register](../common/errs/http_code.go) method.

## Phases of Plugin Execution

Nginx divides requests into 11 phases [reference](https://cloud.tencent.com/developer/article/1377327).

Since tRPC-Gateway is positioned as a business gateway and only operates on the application layer protocol, the plugin
execution phases are not explicitly defined in the core framework like tRPC-Go plugins. Instead, the plugin's internal
logic controls the timing of plugin execution to meet the requirements of most business gateway plugins.

The following code serves as an illustrative example:

```go
// ServerFilter is the server interceptor
func ServerFilter(ctx context.Context, req interface{}, handler filter.ServerHandleFunc) (interface{}, error) {
// It is essential to catch panics to prevent the entire interface from crashing due to exceptions during plugin configuration.
    defer func () {
        if r := recover(); r != nil {
            log.ErrorContextf(ctx, "demo handle panic:%s", string(debug.Stack()))
        }
    }()

    // Perform operations on the request content

    // Perform the actual forwarding operation
    rsp, err := handler(ctx, req)
    if err != nil {
        return nil, utils.ErrWrap(err, "demo plugin handler err")
    }
	// Perform operations on the response content
    withTraceID(ctx)
    // If the operation is performed after the response, it can be implemented using a goroutine to avoid blocking the response
    //
    // go func() {
    // Operations performed after the response, note that the goroutine should not reference the fctx object because it will be reused by fasthttp after the request returns.
    // You can use the fasthttp.Request.CopyTo method to make a copy.
    // }()
    return rsp, nil
}
```

Examples of gateway plugin application scenarios:

- Perform validation, interception, and intervention on requests before executing request forwarding
    - This type of logic can be executed before `handler(ctx,req)`.
- Perform proxy information reporting and response content intervention (modify response headers, response bodies,
  redirects, etc.) after request forwarding is completed
    - This type of logic can be executed during `handler(ctx,req)`.
- Concurrent execution of plugins
    - Non-blocking concurrent execution of plugins can be achieved using goroutines.

## Plugin Error Code Definition

Common gateway plugin errors range from 10000 to 19999. It is recommended for business-specific gateway plugins to avoid
this range.

Already used status codes:
10000: CORS interception
10001: Development environment error
10002: Response transformer error with incorrect JSON value type
10003: Polaris/limiter rate limiting
10004: Batch request error
10005: Batch request upstream response code is non-zero, corresponding to an HTTP status code of 200
10006: Mocking mock response body error
10007: Development environment forwarding error
10008: trpceer2body error to body conversion error

## Business-Specific Plugin Development

The current directory only contains officially maintained common gateway plugins. Each business can create its own
plugin repository as needed and import it into the gateway service instance for registration.

## Special Note

- Do not reference the fasthttp.RequestCtx object in goroutines newly created by plugins because fasthttp will reuse
  this object after the request returns. If the goroutine continues to reference it, unexpected problems may occur.
  Refer to: https://github.com/valyala/fasthttp/issues/146