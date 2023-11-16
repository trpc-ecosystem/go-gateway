# Gateway Core Package

This package contains the core functionality of the gateway, including configuration reading, routing logic, and
protocol forwarding.

### config

Defines the gateway configuration structure and various methods for reading configuration from different sources.

### router

Implements the routing strategy for the gateway. See [路由配置](router/README.md) for more details.

### rule

Implements the rule engine for fine-grained matching rules. See [精细匹配规则引擎](rule/README.md) for more details.

### service

Supports multiple forwarding protocols, currently including HTTP and tRPC. Support for other protocols such as gRPC will
be added in the future.