# tRPC-Gateway

[![LICENSE](https://img.shields.io/badge/license-Apache--2.0-green.svg)](https://github.com/trpc-group/trpc-gateway/blob/main/LICENSE)
## Table of Contents

<!-- TOC -->
* [tRPC-Gateway](#trpc-gateway)
  * [Table of Contents](#table-of-contents)
  * [Introduction](#introduction)
  * [Background](#background)
  * [Gateway Deployment:](#gateway-deployment-)
    * [Running Local Demo](#running-local-demo)
    * [Deployment on with etcd](#deployment-on-with-etcd)
  * [Routing Configuration](#routing-configuration)
  * [Gateway Plugin Development](#gateway-plugin-development)
    * [List of Common Plugins](#list-of-common-plugins)
  * [Forwarding Protocol Support](#forwarding-protocol-support)
      * [Does it support trpc -> trpc protocol forwarding?](#does-it-support-trpc----trpc-protocol-forwarding)
  * [How to Contribute](#how-to-contribute)
<!-- TOC -->

## Introduction

tRPC-Gateway is a business gateway framework in the tRPC ecosystem.

It has the following advantages:

1. Low integration cost: It is essentially a tRPC-Go service, so it can be deployed like any other tRPC-Go service.
2. Low entry barrier: It can fully reuse the tRPC ecosystem, including service governance, monitoring and alerting, protocol support, and log querying.
3. Rich routing strategies: It supports various routing strategies such as exact matching, prefix matching, regular expression matching, fine-grained matching, and gray deployment.
4. Strong scalability: The development threshold for gateway plugins is lowered to the level of developing a tRPC filter, allowing for quick expansion of business-specific gateway logic.
5. Easy enrichment of plugin ecosystem: The development threshold for gateway plugins is low enough to easily enrich the plugin ecosystem.

## Background

In the context of cloud-native technology, business gateways have become indispensable. With the gradual adoption of the internal tRPC framework within the company, there are two solutions to meet the demands for business gateways:
> 1 Modifying an external open-source gateway: This solution allows for the reuse of core functions from an open-source gateway but requires significant adaptation work to fit into the tRPC ecosystem.

> 2 Building a business gateway within the tRPC ecosystem based on the tRPC framework: It was found that this only requires the development of gateway routing logic.

tRPC-Gateway is a business gateway developed based on solution 2.

## Gateway Deployment:

### Running Local Demo

See [example/README.md](example/loader/file/README.md)

### Deployment on with etcd

1. Create a new service repository and add the main.go file, refer to main.go [main.go](example/loader/file/main.go)
2. Apply for Rainbow configuration and add the router.yaml configuration file, refer to [router.yaml](example/loader/etcd/conf/router.yaml)
3. Deploy the service with framework configuration, refer to [trpc_go.yaml](example/loader/etcd/trpc_go.yaml)

## Routing Configuration

Modify the router.yaml file to configure the forwarding of your own interfaces. For more details, see [Routing Configuration](core/router/README.md)

## Gateway Plugin Development

You can extend the functionality of the gateway through plugins. See [Gateway Plugin Development](plugin/README.md) for more details.

### List of Common Plugins

* [x] [CORS](plugin/cors)
* [x] [access log](plugin/accesslog)
* [x] [limiter](plugin/limiter/polaris)
* [x] [request_transformer](plugin/transformer/request)
* [x] [response_transformer](plugin/transformer/response)
* [x] [batch_request](plugin/batchrequest)
* [x] [devenv](plugin/devenv)
* [x] [logreplay](plugin/logreplay)
* [x] [mocking](plugin/mocking)
* [x] [canaryrouter](plugin/polaris/canaryrouter)
* [x] [metarouter](plugin/polaris/metarouter)
* [x] [setrouter](plugin/polaris/setrouter)
* [x] [redirect](plugin/redirect)
* [x] [routercheck](plugin/routercheck)
* [x] [traceid](plugin/traceid)
* [x] [trpcerr2body](plugin/transformer/trpcerr2body)

## Forwarding Protocol Support

tRPC-Gateway supports HTTP as the entry protocol and can forward to other protocols. Custom protocols can be implemented by implementing the [CliProtocolHandler](core/service/protocol/cliprotocol.go) interface.

Supported forwarding protocols:

* http -> http
* http -> trpc
* http -> grpc

#### Does it support trpc -> trpc protocol forwarding?

No, it doesn't support trpc -> trpc protocol forwarding because it would require writing separate plugin logic to support both HTTP and trpc protocols, which would increase complexity.

If the caller expects to use the trpc pb client for calling, you can use the http -> trpc forwarding method. There is no difference in the caller's code, except that the caller's protocol can only use http.

## How to Contribute

You can fork the project, make modifications, and initiate a merge request (MR). For more details, please consult quonliu@tencent.com