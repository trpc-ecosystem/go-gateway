# HTTP to gRPC Forwarding

## Key Implementation Points

- Use the TRPC_GATEWAY_GRPC_HEADER in the context to pass the gRPC request body, response body, and metadata. The main
  purpose is to bypass specific steps in the tRPC-Go framework, such as serialization and deserialization, as the
  grpc-go framework already handles these steps. Skipping these steps reduces unnecessary data conversion.
- Use JSON format to transmit gRPC data. This requires the client to send data in JSON format, and the gRPC server needs
  to support JSON codec by implementing the Encode and Decode methods. Refer to the implementation in the
  grpc-go/encoding package.
- Before the invoke function in tRPC-Go, the JSON request body and headers need to be placed in the grpc header defined
  by TRPC_GATEWAY_GRPC_HEADER. This step essentially performs protocol conversion.
- TRPC_GATEWAY_GRPC_HEADER needs to be placed in the ctx in the server function of the transport layer. This is the
  top-level ctx so that it can be accessed from all places.

## Requirements for Upstream gRPC Services

The upstream gRPC services need to register the JSON codec to support JSON format request bodies.