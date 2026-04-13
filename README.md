# Description

Ngrok like tunneling system created using Go.

## Build

```bash
./build.sh
```

# Usage

```bash
# start tunnel server
tunnel -mode=server -listen=:8080
# start the client
tunnel -mode=agent -server=ws://localhost:8080/ws -url=http://localhost:3000 -subdomain=demo
```
