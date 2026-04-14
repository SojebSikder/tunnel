# Description

Ngrok like tunnel created using Go on quic protocol.

## Build

```bash
./build.sh
```

# Usage

```bash
# start tunnel server
tunnel start-server -listen=:8080
# start http tunnel
tunnel start-agent -server=localhost:8080 -url=http://localhost:3000 -subdomain=demo
# start tcp tunnel (ex. postgres)
tunnel start-agent -server=localhost:8080 -url=localhost:5432 -tcp -remote-port=:5432 -subdomain=mydb
```

## Features
- [x] HTTP tunneling
- [x] TCP tunneling
- [x] dynamic subdomain
