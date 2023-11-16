## Example Instructions

#### Local Execution
- Start etcd and configure router.yaml, refer to [Makefile](./Makefile)
  - Install etcd and execute make run to start an etcd instance locally.
  - Execute make update_router to set route.yaml in etcd.
- Start the demo backend service: Execute the following command in the current directory
```sh
$ make server
```
- Start the gateway service: Open another console and execute the following command in the current directory
```sh
$ make run
```
- Call the API Execute the requests in [test.http](./test.http) and observe the response results.