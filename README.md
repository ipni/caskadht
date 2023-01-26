# :knot: caskadht

`caskadht`, pronounced "Cascade-DHT", is a service that:
* exposes an [IPNI-compatible](https://github.com/ipni/specs/blob/main/IPNI.md#get-multihashmultihash) `GET /multihash/<multihash>` endpoint over HTTP,
* cascades lookup requests over the IPFS Kademlia DHT,
* uses the accelerated DHT client when possible, and
* steams the results back over `ndjson` whenever the request `Accept`s it or regular JSON otherwise.

## Install

To install `caskadht` CLI directly via Golang, run:

```shell
$ go install github.com/ipni/caskadht/cmd/caskadht@latest
```

## Usage

```shell
$ caskadht 
Usage of caskadht:
  -httpListenAddr string
        The caskadht HTTP server listen address. (default "0.0.0.0:40080")
  -libp2pIdentityPath string
        The path to the marshalled libp2p host identity. If unspecified a random identity is generated.
  -libp2pListenAddrs string
        The comma separated libp2p host listen addrs. If unspecified the default listen addrs are used at ephemeral port.
  -logLevel string
        The logging level. Only applied if GOLOG_LOG_LEVEL environment variable is unset. (default "info")
  -useAcceleratedDHT
        Weather to use accelerated DHT client when possible. (default true)
```

### Run Server Locally

To run the `caskadht` HTTP server locally, execute:

```shell
$ go run cmd/caskadht/main.go
```
The above command starts the HTTP API exposed on default listen address: `http://localhost:40080`.
You can then start looking up multihashes, which would cascade onto the DHT. Example:
```shell
$ curl http://localhost:40080/multihash/QmcroxBV9PBPUg2LfeusC25x1C4mckSmQy6hD5rmuvugfj -v
*   Trying 127.0.0.1:40080...
* Connected to localhost (127.0.0.1) port 40080 (#0)
> GET /multihash/QmcroxBV9PBPUg2LfeusC25x1C4mckSmQy6hD5rmuvugfj HTTP/1.1
> Host: localhost:40080
> User-Agent: curl/7.86.0
> Accept: */*
> 
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Connection: Keep-Alive
< Content-Type: application/x-ndjson
< X-Content-Type-Options: nosniff
< Date: Thu, 26 Jan 2023 12:17:02 GMT
< Transfer-Encoding: chunked
< 
{"MultihashResults":[{"Multihash":"EiDXvX3xT1nGu2ZNdAM0rjm9g+/GAyirmnV5MfAVHoPLsg==","ProviderResults":[{"ContextID":"aXBmcy1kaHQtY2FzY2FkZQ==","Metadata":"gBI=","Provider":{"ID":"12D3KooWFRY4zb9Yvh7Pm5itdpVogtw2XDs68VgKwF1SkEy7eiEC","Addrs":[]}}]}]}

```

To shutdown the server, interrupt the terminal by pressing `Ctrl + C`

## License

[SPDX-License-Identifier: Apache-2.0 OR MIT](LICENSE.md)
