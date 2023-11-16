### Routing Forwarding

#### Normal Forwarding

##### Exact Match

- Direct forwarding
- Rewrite forwarding

##### Prefix Match

- Remove prefix forwarding
- Replace prefix forwarding

##### Regular Expression Match

- Regular expression matching

#### Forwarding Exceptions

##### No Route Matched

- Return 404

##### Backend Service Request Failure

- Monitoring report:
  - Report as an exception
- Response headers:
  - trpc-fun
  - trpc-err-msg

##### Backend Service Returns trpc err

- Monitoring report: Normal
- Response headers: Trpc-Error-Msg: business err, Trpc-Func-Ret: 11111

##### Backend Service Request Successful, Non-200 Response