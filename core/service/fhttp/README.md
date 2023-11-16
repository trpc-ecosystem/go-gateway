# Implementation of HTTP Forwarding Logic in fasthttp Version

-----

## Why use fasthttp instead of net/http?

The advantages of using the integrated fasthttp framework include higher concurrency support:

- net/http creates a new goroutine for each connection; fasthttp reuses a goroutine with a worker, reducing the pressure
  on the runtime to schedule goroutines.

- net/http parses a lot of request data into map[string]string(http.Header) or map[string][]string(http.Request.Form),
  which involves unnecessary conversions from []byte to string. These can be avoided.

- net/http generates a new *http.Request and http.ResponseWriter for each HTTP request it parses; fasthttp parses HTTP
  data into *fasthttp.RequestCtx and then uses sync.Pool to reuse structure instances, reducing the number of objects.

- fasthttp delays parsing data in HTTP requests, especially the Body part. This saves a lot of consumption in cases
  where the Body is not directly operated on.

**Working principle of net/http**

![](https://tonybai.com/wp-content/uploads/server-side-performance-nethttp-vs-fasthttp-2.png)

**Working principle of fasthttp**

![](https://tonybai.com/wp-content/uploads/server-side-performance-nethttp-vs-fasthttp-3.png)

## 1.2 Protocol Conversion

### HTTP -> HTTP Forwarding

- Headers in the response will be formatted by default
    - Example: "my-header" will be formatted to "My-Header". Refer
      to [fasthttp header formatting](https://github.com/valyala/fasthttp/blob/f0865d4aabbbea51a81d56ab31a3de2dfc5a9b05/server.go#L334)。
    - Impact: This may affect some features that depend on response headers.
    - Solution: Considering that formatting provides a lot of convenience for header operations and is more in line with
      standards, and non-standard headers are a small scenario, we choose to retain this feature. For necessary
      scenarios, you can use [response_transformer](../../../plugin/transformer/response) to rename the header to the
      target response header format.
- Handling of oversized response headers

When the fasthttp client reads the response header, it uses a read buffer. The default size is 4096 bytes. If the
response header is too large, the following related exception will be reported:

```text
small read buffer. Increase ReadBufferSize
```

You can overwrite the [DefaultClientReadBufferSize](transport.go) variable in the main.go of the gateway instance
project to modify the size of the read buffer.

Note: Do not set it too large, as this will increase memory consumption.

### HTTP -> tRPC Forwarding

- HTTP request headers will be transparently passed to the tRPC interface through metadata.

## Introduction to tRPC-Gateway Response Headers

- X-Proxy-Latency: The latency of the gateway proxy logic.
- X-Upstream-Latency: The latency of the upstream.
- Server: tRPC-Gateway: Indicates that it is a tRPC-Gateway proxy.

## 1.4 Pre-routing Interception of Illegal Requests

Since the tRPC-Gateway route matching is performed before the plugin execution, it is impossible to intercept illegal
requests through the plugin to avoid unnecessary route matching and forwarding.

For example, there are often such illegal requests in the news online. Because the /share/ prefix match is used,
forwarding will also be performed, and the Galileo plugin will also report, and there will be a large number of such
invalid interfaces in the interface dimension:

- /share/upfile.asp
- /share/tz.php
- /share/UPGRADE.md

Therefore, a pre-interception function that the business side can overwrite is provided to implement the interception of
invalid requests before route matching. Just overwrite the core/service/fhttp.DefaultReqValidate method.

Note: This is a high-risk operation. The interception logic added must be confirmed not to intercept normal interfaces.
You can capture valid interfaces online through monitoring and verify the correctness of the interception logic through
unit tests.

## 1.5 Monitoring and Alerting

The gateway has added the following metrics reporting:

- Reporting of failed route configuration updates: trpc_gateway_report.reload_router_err_count
- Reporting of errors for 404 requests, with error code: 1002
- Reporting of errors for non-200 upstream responses, with error code: 1008, along with the HTTP status code and method
- Reporting of other gateway errors, see [error code definitions](../../../common/errs/errs.go)

All of the above reporting can be configured for corresponding monitoring and alerting, to differentiate between gateway
logic errors and upstream service errors.

You can also override the DefaultReportErr method to implement custom error reporting.

## 1.6 Handling of err Returns in trpc Interfaces

Consider the two common ways of handling err in trpc interfaces:

- Not returning err: This is for client-facing interfaces, where trpc interfaces generally pass business information
  through the response body's error code, and do not return err. Instead, they uniformly return nil. This approach is
  more user-friendly for clients.

```go
// SayHello trpc interface implementation
func (s *greeterServiceImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
hiRsp, err := s.proxy.SayHi(ctx, req, opts...)
    if err != nil {
        log.ErrorContextf(ctx, "say hi err:%s", err)
        // trpc interface uniformly returns nil, and the calling party determines the business status code through rsp.Code
        rsp := &pb.HelloReply{
            Code: 10000,
            Msg:  "say hi err",
		}
        return rsp, nil
    }
    rsp := &pb.HelloReply{
        Msg: "Hello",
    }
    return rsp, nil
}
```

- Returning err: This is for non-client-facing interfaces, where err is directly returned, and the calling party
  determines the business error through the err code. This approach is more user-friendly for trpc clients.

```go
// SayHello trpc interface implementation
func (s *greeterServiceImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
hiRsp, err := s.proxy.SayHi(ctx, req, opts...)
    if err != nil {
        log.ErrorContextf(ctx, "say hi err:%s", err)
        // Return err, and the calling party determines the business status code through err.Code
        return rsp, err
    }
    rsp := &pb.HelloReply{
        Msg: "Hello",
    }
    return rsp, nil
}
```

tRPC-Gateway handles the following to accommodate both client-facing and trpc client calling:

- trpc interfaces do not return err:
    - Whether forwarding HTTP interfaces or trpc interfaces, the gateway forwards them as-is.
- trpc interfaces return err:
    - The gateway responds with an HTTP status code of 200, and the err code and err msg are encapsulated in the
      Trpc-Func-Ret and Trpc-Error-Msg response headers, respectively. This aligns with the response format of trpc's
      HTTP interfaces. For trpc clients, the calling party can retrieve the error information through the err code.
    - If you want to encapsulate the trpc err in the response body for client use, you can assemble the response body
      through plugins.
