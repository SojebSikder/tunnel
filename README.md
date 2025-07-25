# Description

Ngrok like tunneling system created using Go.

## Build

```
./build.sh
```

# Usage

```bash
# start tunnel server
tunnel start-server --port 7000
# start the client
tunnel tunnel --url http://localhost:4000 --host http://localhost:7000
```

## Supported commands

- `start-server` - start the tunnel server
- `tunnel` - start the client,
  - `--url` flag for speicify the client app url -`--host` to speicify the tunnel server host address

## Tests

```bash
go test ./...
# with benchmark
go test ./... -bench=.
# with coverage
go test -cover ./...
```
