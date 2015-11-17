# Manual tests

For these manual tests, you'll need 3 servers (*client*, *proxy*, *origin*), preferably in the same network so you can reach the highest possible transfer rates and smaller latencies.  Any suggestion to improve these tests is welcome.



## Monitoring strategies and tools

### General

Look at DigitalOcean's graph data (Disk, CPU, Bandwidth).

### Pprof

The server implements a [pprof](https://golang.org/pkg/net/http/pprof/) profiling server, with customizable port.

### Some tools to keep in mind

Memory monitoring

```
watch -n 0.5 'pmap <process-pid> | tail -n 1'
```

Connection monitoring

```
nethogs
```

## Stress tests and benchmarks

### Siege

Installation

```
git clone uaalto:JoeDog/siege.git
cd siege
autoreconf
./configure && make && sudo make install
```

Configuration file can be found in the tests folder

```
cp test/.siegerc ~
```

Run it with:

```
PROXY_HOST=128.199.35.205 PROXY_PORT=8080 siege www.google.com
```

The IP and port correspond to the server where you set up the proxy. Try the `-c` flag to use more concurrent users (10 is the default). Tests have been carried with:

* 10 concurrent users
* 100 concurrent users
* 400 concurrent users


### Boom

Install (needs Go)

```
go get github.com/rakyll/boom
```

Test proxying to Google:

```
boom -allow-insecure -n 400 -c 20 -q 100 -x http://<proxy-ip>:8080 https://www.google.com/humans.txt
```

You can use higher concurrency (`-c`) and higher number of requests (`-n`).


## Test large file transmission

### Download a large file with fast origin link and fast client link

#### Download with unbounded transfer rates

In client machine:

```
wget -e use_proxy=yes -e http_proxy=<proxy-addr>:8080 http://releases.ubuntu.com/15.10/ubuntu-15.10-desktop-amd64.iso
```

In proxy machine: use memory footprint monitoring and connection monitoring.

#### Download with client rate limit

In client machine:

```
wget --limit-rate=20k -e use_proxy=yes -e http_proxy=<proxy-addr>:8080 http://releases.ubuntu.com/15.10/ubuntu-15.10-desktop-amd64.iso
```

In proxy machine: use memory footprint monitoring and connection monitoring.

#### Download with origin rate limit

<TODO>

#### Upload with unbounded transfer rates

<TODO>

#### Upload with client rate limit

<TODO>

#### Upload with origin rate limit

<TODO>

#### Tests with multiple files simultaneously

Force multiple connections to be opened, by downloading multiple times. Using `wget` will rename automatically, so many independent simultaneous downloads can be performed.
