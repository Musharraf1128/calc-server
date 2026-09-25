# calc-server

A minimal HTTP/1.1 calculator server, written in Go using only the raw
`net` package (no HTTP framework). Built for a network architecture
course assignment focused on persistent-connection handling.

Name: Shah Musharaf ul islam
Roll No: 10447

## What it does

Serves four arithmetic operations over GET requests, and returns the
correct HTTP status codes for the documented error cases:

| Request                  | Response         |
|---------------------------|------------------|
| `GET /add?a=2&b=3`        | `200` → `5`      |
| `GET /sub?a=10&b=4`       | `200` → `6`      |
| `GET /mul?a=6&b=7`        | `200` → `42`     |
| `GET /div?a=9&b=3`        | `200` → `3`      |
| `GET /div?a=1&b=0`        | `400` (divide by zero) |
| `GET /add?a=x&b=3`        | `400` (non-numeric input) |
| `GET /pow?a=2&b=8`        | `404` (unknown route) |
| `POST /add`                | `405` (method not allowed) |
| `GET /add` with no `Host`  | `400` (missing required header) |

## The constraint that matters

The server keeps the TCP connection open across requests. Every
request is read off the socket by consuming exactly its bytes (request
line + headers up to the blank line, plus any declared body) so that
back-to-back requests on the same connection parse correctly — no
request bleeds into the next, and the connection is never closed after
a single exchange.

## Running it

```bash
go build -o calc-server main.go
./calc-server
# listens on :8080
```

## Testing manually

```bash
curl "http://localhost:8080/add?a=2&b=3"

(or)

run: python/python3 test_server.py
```

To verify persistence, open one raw socket and send multiple requests
on it sequentially (e.g. with Python's `socket` module) — the
connection should stay open and every response arrive independently.
