# trpc-gateway路由模块

- [路由配置方式](#路由配置方式)
- [路由配置详解](#路由配置详解)
    - [路由项配置router](#路由项配置router)
        - [method](#method)
        - [is_regexp](#isregexp)
        - [id](#id)
        - [rewrite](#rewrite)
        - [strip_path](#strippath)
        - [report_method](#reportmethod)
        - [target_service](#targetservice)
        - [hash_key](#hashkey)
        - [host](#host)
        - [plugins](#路由插件)
        - [rule](#rule)
    - [后端服务配置client](#client)
    - [全局插件配置plugins](#全局插件)
- [网关插件执行](#网关插件执行)

# 路由逻辑
![img.png](docs/router.png)

### 匹配顺序

精确匹配 -> 前缀匹配 -> 正则匹配 -> 精细匹配

多个路径相同的路由项，按照配置顺序匹配

【Attention】注意与 nginx 匹配规则的区别：

- nginx 先匹配 host，再匹配 host 下的 path
- tRPC-Gateway 先匹配 path，再匹配 host

【Attention】所有的精细匹配失败之后，不会再进行其他匹配！

场景举例：

请求： http://r.inews.qq.com/user/info

有如下的路由项：

- method: /user/info host: f.inews.qq.com
- method: /user/info host: w.inews.qq.com
- method: /user/ host:

首先根据 URI 匹配了两个精确匹配的路由项：

- method: /user/info host: f.inews.qq.com
- method: /user/info host: w.inews.qq.com

然后继续根据 host 进行精确匹配，这时候发现都没有匹配上，就会返回 404。

而不会前缀匹配到 /user/

- method: /user/ host:

# 路由配置方式

路由配置支持：

- 本地文件
    - 设置 trpc.yaml 文件中 global.conf_provider=file 参考 [trpc.yaml](../../example/loader/file/trpc_go.yaml)
    - 通过启动参数 --router={you_router_conf} 指定配置文件
- 配置中心, 当前支持 Etcd
    - Etcd
        - 参考 [etcd loader](../loader/etcd/README.md)

# 路由配置详解

配置分为三个部分 router：转发规则配置 client：转发的后端服务配置 plugins：全局插件配置

采用 yaml 格式进行网关路由的配置。完整配置示例参考 [router.yaml](../../example/loader/file/conf/router.yaml)

### 路由项配置router

router 配置是一个数组，数组的每一个元素是一个路由转发规则项。
示例配置：

```yaml
router: # 路由配置
  - method: ^/v1/user/ # 正则路由
    is_regexp: true  # 是否是正则路由，设置为 true 才会进行正则匹配
    id: "path:^/v1/user/" # 路由 id，用来标识一个路由，方便调试。（ method 会重复）
    rewrite: /v1/user/info # 重写路径
    strip_path: false
    target_service: # 上游服务
      - service: trpc.user.service # 服务名称，对应 client 配置里的 name
        weight: 10 # 服务权重，权重之和不可为0
        rewrite: /search # 服务级别重写路径，优先级高于 router 级别；优先级高于 strip_path 配置
        strip_path: false # 是否去除前缀
    hash_key: ""
    host:
      - test.shizi.qq.com # host，匹配之后才会命中当前路由。为空则匹配所有请求的 host
    plugins: # 路由级别插件
      - name: demo # 插件名称，必填
        type: gateway
        props: # 插件属性
          suid_name: suidxxx
    rule:
      conditions:
        - key: devid
          val: xxxx,yyyyy
          oper: in
      expression: "0"
```

每个字段含义解释如下：

--------

#### method

path匹配规则，类型为 string，有如下三种匹配规则

- 精确匹配：path 和 method 完全一样。如请求 path 为 /user/info 会匹配到 method = /user/info 的路由项
- 前缀匹配：path 包含 method 前缀。如请求 path 为 /user/info 会匹配到 method = /user/ 的路由项
- 正则匹配：path 命中 method 的正则规则。如请求 path 为 /user/info 会匹配到 method = (/user/info|/user/add) 且 is_regexp =
  true 的路由项

三种匹配逻辑的优先级为：精确匹配 > 前缀匹配 > 正则匹配

当 method 重复时，即匹配到多个路由项，会尝试通过 rule（详见 [rule](#rule)）进行精细匹配，会返回匹配到的第一个路由项。 如果
rule 也没有匹配到，则会返回第一个没有配置 rule 的路由项

--------

#### is_regexp

是否是正则匹配，类型为 bool，默认为 false，为 true 标识当前是正则路由。 因为正则路由是通过遍历进行匹配的，显示标识正则路由可以提高匹配效率

--------

#### id

路由的唯一标识，用来标识一个路由项，用来开发调试。

--------

#### rewrite

重写的路径，分为精确路径和前缀路径

- 精确路径：即不以 / 结尾，优先级最高。
    - 如客户端请求为 /user/info ,需要转发到 /v1/user/info，则配置 rewrite=/v1/user/info。 不配置则依旧转发到 /user/info
- 前缀路径：以 / 结尾
    - 如客户端请求为 /user/info ,需要转发到 /v1/user/info，则配置 rewrite=/v1/。 不配置则依旧转发到 /user/info

--------

#### strip_path

转发时是否去除前缀

如请求 path = /v1/user/info，预期转发到 /user/info，则可配置 method = /v1/， strip_path = true 。 优先级低于 rewrite 的精确路径配置

可以结合 rewrite 的前缀路径配置，实现接口前缀重写。

如请求 path = /v1/user/info，预期转发到 /v2/user/info，则可配置 method = /v1/， strip_path = true，rewrite = /v2/

--------

#### report_method

只上报 method，不上报 path，防止类似 /a/{article_id} 的接口,造成被调接口监控维度爆炸

--------

#### target_service

目标服务配置，是一个数组，可以通过每个元素的 weight 字段配置流量权重

#### target_service.service

服务名称，对应 client 配置里的 name 字段

#### target_service.weight

流量权重，多个 service 的 weight 之和需要 > 0。只有一个 service 可以不配置

#### target_service.rewrite

service 级别的 rewrite，逻辑同 router 级别 rewrite

#### target_service.strip_path

service 级别的 strip_path，逻辑同 router 级别 strip_path

--------

#### hash_key

配合多个 target_service 配置使用。如配置了 devid，则请求参数（依次判断 query 参数，header，cookie）包含相同 devid 的请求，都会路由到同一个
target_service

--------

#### host

目标请求的 host 列表，在当前集合中才会匹配到当前路由项。为空则匹配所有 host

--------

#### 路由插件

路由级别插件配置，是一个数组，可配置多个插件。只对当前路由项生效

#### plugins[0].name

插件名称，必填项。需要和插件定义里的name相同

#### plugins[0].type

插件类型，非必填，默认为 gateway，需要和插件定义里的type相同

#### plugins[0].props

插件属性，非必填。每个插件都可以有自己的配置字段

--------

#### rule

精细匹配规则，通过请求参数进行路由匹配。

#### rule.conditions

规则匹配列表

#### rule.conditions[0].key

请求参数名称，如配置了 devid，会依次查询query参数，header参数，cookie 里，名称为 devid 的参数。

业务方可以通过重写 core/router.DefaultGetString 方法，定制化自己的参数获取逻辑，如：获取 json 请求体里的参数。

#### rule.conditions[0].val

请求参数的值，如匹配 devid 为 xxx 的请求

#### rule.conditions[0].oper

逻辑运算符，支持以下操作

| 操作符 | -描述-- |      备注      |
|:---:|:-----:|:------------:|
| ==  |  等于   |              |
| !=  |  不等于  |              |
| \>  |  大于   |              |
| > = | 大于等于  |              |
|  <  |  小于   |              |
| <=  | 小于等于  |              |
| in  | 在集合中  | val值用 ',' 分隔 |
| !in | 不在集合中 | val值用 ',' 分隔 |

#### rule.conditions[0].expression

conditions 逻辑运算，支持|| 和 && 操作。如 0&&1||2 表示条件 conditions[0] 且 conditions[1] 或 conditions[2] 条件满足时，命中当前路由项

--------

### client

client 配置基于 trpc client 逻辑开发，配置字段和逻辑基本和 trpc client
一致。

配置示例：

```yaml
client:
  - name: trpc.inews.user.User
    disable_servicerouter: false
    namespace: Production
    target: polaris://trpc.inews.user.User
    network: tcp
    timeout: 500
    protocol: fasthttp # 目前只支持 fasthttp 和 trpc，其中 fasthttp 即为 http 协议
    serialization: null
    plugins: # 服务级别插件
      - name: auth
        type: gateway
        props:
```

有如下差异：

- protocol 目前只支持 fasthttp 和 trpc，其中 fasthttp 即为 http 协议
- 增加了 plugins 字段，可以配置服务级别的网关插件。配置方式同全局插件配置
- 禁用filter配置

--------

### 全局插件

全局插件配置，对所有的请求生效。

配置示例

```yaml
plugins:
  - name: demo # 插件名称，必填
    type: gateway # 插件类型，不填默认为 gateway，需要与插件定义里的 type 保持一致
    props: # 插件属性
      suid_name: suidxxx
```

#### plugins[0].name

插件名称，必填项。需要和插件定义里的name相同

#### plugins[0].type

插件类型，非必填，默认为 gateway，需要和插件定义里的type相同

#### plugins[0].props

插件属性，非必填。每个插件都可以有自己的配置字段

# 网关插件执行

网关插件有个配置位置：全局插件(plugins)、服务插件(client[0].plugins)、路由插件(router[0].plugins)

执行顺序为：全局插件 > 服务插件 > 路由插件

当插件重复时，按照就近原则，配置优先级为：路由插件 > 服务插件 > 全局插件

网关插件开发、注册，请参考 [网关插件开发](../../plugin/README.md)
