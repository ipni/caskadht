# :knot: caskadht

`caskadht`, pronounced "Cascade-DHT", is a service that:

* exposes:
    * `GET /routing/v1/providers/<cid>` compatible with
      IPFS [HTTP delegated routing](https://github.com/ipfs/specs/pull/337), and
    * `GET /multihash/<multihash>` compatible
      with [IPNI HTTP query API](https://github.com/ipni/specs/blob/main/IPNI.md#get-multihashmultihash)
* cascades lookup requests over the IPFS Kademlia DHT,
* uses the accelerated DHT client when possible, and
* steams the results back over `ndjson` whenever the request `Accept` header permits it, or
  non-streaming JSON otherwise.

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
You can then start looking up multihashes, which would cascade onto the DHT.

To shutdown the server, interrupt the terminal by pressing `Ctrl + C`

#### Example IPNI `ndjson` response:

```text
$ curl http://localhost:40080/multihash/QmfQJymEUXsGNzHMmpGYmUcFiAtGw2ia97EXNDVDZbZjgm -v --max-time 1
*   Trying 127.0.0.1:40080...
* Connected to localhost (127.0.0.1) port 40080 (#0)
> GET /multihash/QmfQJymEUXsGNzHMmpGYmUcFiAtGw2ia97EXNDVDZbZjgm HTTP/1.1
> Host: localhost:40080
> User-Agent: curl/7.86.0
> Accept: */*
> 
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Connection: Keep-Alive
< Content-Type: application/x-ndjson
< X-Content-Type-Options: nosniff
< Date: Sat, 28 Jan 2023 18:06:44 GMT
< Transfer-Encoding: chunked
< 
{"ContextID":"aXBmcy1kaHQtY2FzY2FkZQ==","Metadata":"gBI=","Provider":{"ID":"12D3KooWSnniGsyAF663gvHdqhyfJMCjWJv54cGSzcPiEMAfanvU","Addrs":["/ip6/2604:1380:45f1:8400::1/tcp/4002/ws","/ip4/145.40.89.195/tcp/4002/ws","/ip6/2604:1380:45f1:8400::1/tcp/4001","/ip4/145.40.89.195/tcp/4001"]}}

{"ContextID":"aXBmcy1kaHQtY2FzY2FkZQ==","Metadata":"gBI=","Provider":{"ID":"12D3KooWEDMw7oRqQkdCJbyeqS5mUmWGwTp8JJ2tjCzTkHboF6wK","Addrs":["/ip6/2604:1380:45e1:2700::3/tcp/4001","/ip6/2604:1380:45e1:2700::3/tcp/4002/ws","/ip4/139.178.68.91/tcp/4001","/ip4/139.178.68.91/tcp/4002/ws"]}}

{"ContextID":"aXBmcy1kaHQtY2FzY2FkZQ==","Metadata":"gBI=","Provider":{"ID":"12D3KooWRgXWwnZQJgdW1GHW7hJ5UvZ8MLp7HBCSWS596PypAs8M","Addrs":["/ip4/147.75.49.91/tcp/4002/ws","/ip6/2604:1380:45e1:2700::b/tcp/4001","/ip6/2604:1380:45e1:2700::b/tcp/4002/ws","/ip4/147.75.49.91/tcp/4001"]}}

* Operation timed out after 1005 milliseconds with 1890 bytes received
* Closing connection 0
curl: (28) Operation timed out after 1005 milliseconds with 1890 bytes received
```

#### Example IPFS Delegated Routing `ndjson` Response

```text
$ curl  http://localhost:40080/routing/v1/providers/QmfQJymEUXsGNzHMmpGYmUcFiAtGw2ia97EXNDVDZbZjgm -v --max-time 1
*   Trying 127.0.0.1:40080...
* Connected to localhost (127.0.0.1) port 40080 (#0)
> GET /routing/v1/providers/QmfQJymEUXsGNzHMmpGYmUcFiAtGw2ia97EXNDVDZbZjgm HTTP/1.1
> Host: localhost:40080
> User-Agent: curl/7.86.0
> Accept: */*
> 
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Connection: Keep-Alive
< Content-Type: application/x-ndjson
< X-Content-Type-Options: nosniff
< Date: Sat, 28 Jan 2023 18:07:40 GMT
< Transfer-Encoding: chunked
< 
{"Protocol":"transport-bitswap","Schema":"bitswap","ID":"12D3KooWHVXoJnv2ifmr9K6LWwJPXxkfvzZRHzjiTZMvybeTnwPy","Addrs":["/ip4/145.40.89.101/tcp/4001","/ip4/145.40.89.101/tcp/4002/ws","/ip4/145.40.89.101/udp/4001/quic","/ip6/2604:1380:45f1:d800::1/tcp/4001","/ip6/2604:1380:45f1:d800::1/tcp/4002/ws","/ip6/2604:1380:45f1:d800::1/udp/4001/quic"]}

{"Protocol":"transport-bitswap","Schema":"bitswap","ID":"12D3KooWDpp7U7W9Q8feMZPPEpPP5FKXTUakLgnVLbavfjb9mzrT","Addrs":["/ip4/147.75.80.75/tcp/4001","/ip4/147.75.80.75/tcp/4002/ws","/ip4/147.75.80.75/udp/4001/quic","/ip6/2604:1380:4601:f600::5/tcp/4001","/ip6/2604:1380:4601:f600::5/tcp/4002/ws","/ip6/2604:1380:4601:f600::5/udp/4001/quic"]}

{"Protocol":"transport-bitswap","Schema":"bitswap","ID":"12D3KooWCrBiagtZMzpZePCr1tfBbrZTh4BRQf7JurRqNMRi8YHF","Addrs":["/ip4/147.75.87.65/tcp/4001","/ip4/147.75.87.65/tcp/4002/ws","/ip4/147.75.87.65/udp/4001/quic","/ip6/2604:1380:4601:f600::1/tcp/4001","/ip6/2604:1380:4601:f600::1/tcp/4002/ws","/ip6/2604:1380:4601:f600::1/udp/4001/quic"]}

{"Protocol":"transport-bitswap","Schema":"bitswap","ID":"12D3KooWRNijznEQoXrxBeNLb2TqbSFm8gG8jKtfEsbC1C9nPqce","Addrs":["/ip4/147.75.87.211/tcp/4001","/ip4/147.75.87.211/tcp/4002/ws","/ip4/147.75.87.211/udp/4001/quic","/ip6/2604:1380:4601:f600::3/tcp/4001","/ip6/2604:1380:4601:f600::3/tcp/4002/ws","/ip6/2604:1380:4601:f600::3/udp/4001/quic"]}

* Operation timed out after 1001 milliseconds with 1378 bytes received
* Closing connection 0
curl: (28) Operation timed out after 1001 milliseconds with 1378 bytes received
```

## License

[SPDX-License-Identifier: Apache-2.0 OR MIT](LICENSE.md)
