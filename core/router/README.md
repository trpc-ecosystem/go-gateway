<!-- TOC -->
* [tRPC-Gateway Routing Module](#trpc-gateway-routing-module)
    * [Matching order](#matching-order)
* [Routing Configuration Methods](#routing-configuration-methods)
* [Routing Configuration Details](#routing-configuration-details)
    * [Router Configuration](#router-configuration)
      * [method](#method)
      * [is_regexp](#isregexp)
      * [id](#id)
      * [rewrite](#rewrite)
      * [strip_path](#strippath)
      * [report_method](#reportmethod)
      * [target_service](#targetservice)
      * [target_service.service](#targetserviceservice)
      * [target_service.weight](#targetserviceweight)
      * [target_service.rewrite](#targetservicerewrite)
      * [target_service.strip_path](#targetservicestrippath)
      * [hash_key](#hashkey)
      * [host](#host)
      * [Route Plugins](#route-plugins)
      * [plugins[0].name](#plugins-0-name)
      * [plugins[0].type](#plugins-0-type)
      * [plugins[0].props](#plugins-0-props)
      * [rule](#rule)
      * [rule.conditions](#ruleconditions)
      * [rule.conditions[0].key](#ruleconditions-0-key)
      * [rule.conditions[0].val](#ruleconditions-0-val)
      * [rule.conditions[0].oper](#ruleconditions-0-oper)
      * [rule.conditions[0].expression](#ruleconditions-0-expression)
    * [client](#client)
    * [Global Plugins](#global-plugins)
      * [plugins[0].name](#plugins-0-name-1)
      * [plugins[0].type](#plugins-0-type-1)
      * [plugins[0].props](#plugins-0-props-1)
* [Execution of Gateway Plugins](#execution-of-gateway-plugins)
<!-- TOC -->
# tRPC-Gateway Routing Module

### Matching order

Exact match -> Prefix match -> Regex match -> Fine-grained match

For multiple routes with the same path, they are matched in the order of configuration.

【Attention】Note the difference in matching rules compared to nginx:

- Nginx first matches the host and then matches the path under the host.
- tRPC-Gateway first matches the path and then matches the host.

【Attention】After all fine-grained matches fail, no further matching will be performed!

Example scenario:：

Request: http://r.inews.qq.com/user/info

There are the following route items:

- method: /user/info host: f.inews.qq.com
- method: /user/info host: w.inews.qq.com
- method: /user/ host:

First, two exact matches are made based on the URI:

- method: /user/info host: f.inews.qq.com
- method: /user/info host: w.inews.qq.com

Then, a precise match is made based on the host, but none of them match, so a 404 response is returned.

Prefix matching to /user/ will not occur.

- method: /user/ host:

# Routing Configuration Methods

Routing configuration supports:

- Local file
    - Set global.conf_provider=file in the trpc.yaml file, refer to [trpc.yaml](../../example/loader/file/trpc_go.yaml)
    - Specify the configuration file through the startup parameter --router={your_router_conf}
- Configuration center, currently supports Etcd
    - Etcd
        - refer to [etcd loader](../loader/etcd/README.md)

# Routing Configuration Details

The configuration is divided into three parts: router for forwarding rule configuration, client for backend service
configuration, and plugins for global plugin configuration.

The gateway routing configuration is done using the YAML format. For a complete configuration example, refer
to [router.yaml](../../example/loader/file/conf/router.yaml)

### Router Configuration

The router configuration is an array, where each element represents a routing forwarding rule item. Here is an example
configuration:

```yaml
router: # Router configuration
  - method: ^/v1/user/ # Regex route
    is_regexp: true  # Whether it is a regex route, set to true for regex matching
    id: "path:^/v1/user/" # Route ID, used to identify a route for debugging (method can be duplicated)
    rewrite: /v1/user/info # Rewrite path
    strip_path: false
    target_service: # Upstream service
      - service: trpc.user.service # Service name, corresponding to the name in the client configuration
        weight: 10 # Service weight, the sum of weights cannot be 0
        rewrite: /search # Service-level rewrite path, higher priority than router-level; higher priority than strip_path configuration
        strip_path: false # Whether to strip the prefix
    hash_key: ""
    host:
      - test.shizi.qq.com # Host, only matches the current route if matched. If empty, matches all request hosts
    plugins: # Route-level plugins
      - name: demo # Plugin name, required
        type: gateway
        props: # Plugin properties
          suid_name: suidxxx
    rule:
      conditions:
        - key: devid
          val: xxxx,yyyyy
          oper: in
      expression: "0"
```

Each field is explained as follows:

--------

#### method

Path matching rule, type string, with the following three matching rules:

- Exact match: The path and method are exactly the same. For example, if the request path is /user/info, it will match
  the route item with method = /user/info.
- Prefix match: The path contains the prefix of the method. For example, if the request path is /user/info, it will
  match the route item with method = /user/.
- Regex match: The path matches the regex rule of the method. For example, if the request path is /user/info, it will
  match the route item with method = (/user/info|/user/add) and is_regexp = true.

The priority of the three matching logics is: Exact match > Prefix match > Regex match.

When there are duplicate methods, meaning multiple route items match, it will try to perform fine-grained matching using
the rule (see [rule](#rule)), and return the first matched route item. If the rule does not match either, it will return
the first route item that does not have a rule configured.

--------

#### is_regexp

Whether it is a regex match, type bool, default is false. If true, it indicates that it is a regex route. Since regex
routes are matched through iteration, explicitly marking it as a regex route can improve matching efficiency.

--------

#### id

The unique identifier of the route, used to identify a route item for development and debugging.

--------

#### rewrite

The rewritten path, divided into exact path and prefix path.

- Exact path: It does not end with /. It has the highest priority.
    - For example, if the client request is /user/info and it needs to be forwarded to /v1/user/info, then configure
      rewrite=/v1/user/info. If not configured, it will still be forwarded to /user/info.
- Prefix path: It ends with /.
    - For example, if the client request is /user/info and it needs to be forwarded to /v1/user/info, then configure
      rewrite=/v1/. If not configured, it will still be forwarded to /user/info.

--------

#### strip_path

Whether to remove the prefix when forwarding.

For example, if the request path is /v1/user/info and it is expected to be forwarded to /user/info, you can configure
method = /v1/ and strip_path = true. The priority is lower than the exact path configuration of rewrite.

It can be combined with the prefix path configuration of rewrite to achieve prefix rewriting of interfaces.

For example, if the request path is /v1/user/info and it is expected to be forwarded to /v2/user/info, you can configure
method = /v1/, strip_path = true, and rewrite = /v2/.

--------

#### report_method

Only report the method of router, not the request path, to prevent explosion of monitoring dimensions for path like
/a/{article_id}.

--------

#### target_service

Target service configuration, which is an array. The weight field of each element can be used to configure traffic
weight.

#### target_service.service

Service name, corresponding to the name field in the client configuration.

#### target_service.weight

Traffic weight, the sum of weights of multiple services needs to be > 0. Only one service can be configured without
weight.

#### target_service.rewrite

Service-level rewrite, the logic is the same as the rewrite at the router level.

#### target_service.strip_path

Service-level strip_path, the logic is the same as the strip_path at the router level.

--------

#### hash_key

Used in conjunction with multiple target_service configurations. For example, if devid is configured, requests that
contain the same devid in the request parameters (query parameters, headers, cookies) will be routed to the same
target_service.

--------

#### host

The list of target request hosts. The current route item will only match if the host is in this list. If empty, it
matches all hosts.

--------

#### Route Plugins

Route-level plugin configuration, which is an array and can have multiple plugins. Only applicable to the current route
item.

#### plugins[0].name

Plugin name, required. It needs to be the same as the name defined in the plugin definition.

#### plugins[0].type

Plugin type, optional. Default is gateway. It needs to be the same as the type defined in the plugin definition.

#### plugins[0].props

Plugin properties, optional. Each plugin can have its own configuration fields.

--------

#### rule

Fine-grained matching rules for routing based on request parameters.

#### rule.conditions

List of rule conditions.

#### rule.conditions[0].key

Request parameter name. For example, if devid is configured, it will search for the parameter named devid in the query
parameters, headers, and cookies.

Service providers can customize their own parameter retrieval logic by overriding the core/router.DefaultGetString
method, such as retrieving parameters from the JSON request body.

#### rule.conditions[0].val

Value of the request parameter. For example, matching requests with devid equal to xxx.

#### rule.conditions[0].oper

逻辑运算符，支持以下操作

| Operator |    Description    |              Note               |
|:--------:|:-----------------:|:-------------------------------:|
|    ==    |       Equal       |                                 |
|    !=    |    Not equal	     |                                 |
|    >     |      Greater      |                                 |
|    >=    | Greater or equal	 |                                 |
|    <     |       Less        |                                 |
|    <=    |  Less or equal	   |                                 |
|    in    |      In set	      | val values are separated by ',' |
|   !in    |    Not in set	    | val values are separated by ',' |
|  regexp  |   Regex match	    |                                 |

#### rule.conditions[0].expression

Logical operators for conditions, supporting || (OR) and && (AND) operations. For example, 0&&1||2 means that the
current route item will be matched when conditions[0] AND conditions[1] OR conditions[2] are satisfied.

--------

### client

The client configuration is based on the trpc client logic, and the configuration fields and logic are mostly consistent
with the trpc client. 

Configuration example：

```yaml
client:
  - name: trpc.inews.user.User
    disable_servicerouter: false
    namespace: Production
    target: polaris://trpc.inews.user.User
    network: tcp
    timeout: 500
    protocol: fasthttp # Currently, the supported protocols are fasthttp, trpc, and grpc. Among them, fasthttp represents the HTTP protocol.
    serialization: null
    plugins: # Service-level plugins
      - name: auth
        type: gateway
        props:
```

There are the following differences with tRPC client:

- The protocol field currently supports fasthttp, where fasthttp represents the HTTP protocol.
- The plugins field has been added, which allows configuring gateway plugins at the service level. The configuration is
  the same as the global plugin configuration.
- The disable_filter configuration has been removed.

--------

### Global Plugins

Global plugin configuration that applies to all requests.

Configuration example:

```yaml
plugins:
  - name: demo # Plugin name, required
    type: gateway # Plugin type, optional. Default is gateway, should match the type defined in the plugin definition
    props: # Plugin properties
      suid_name: suidxxx
```

#### plugins[0].name

Plugin name, required. It should be the same as the name defined in the plugin definition.

#### plugins[0].type

Plugin type, optional. Default is gateway. It should match the type defined in the plugin definition.

#### plugins[0].props

Plugin properties, optional. Each plugin can have its own configuration fields.

# Execution of Gateway Plugins

Gateway plugins can be configured at three levels: global plugins (plugins), service plugins (client[0].plugins), and
route plugins (router[0].plugins).

The execution order is: global plugins > service plugins > route plugins.

When plugins are duplicated, the priority is determined by proximity: route plugins > service plugins > global plugins.

For information on developing and registering gateway plugins, please refer to [Gateway plugin development](../../plugin/README.md)