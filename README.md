# calc-server

A minimal HTTP/1.1 calculator server, written in Go using only the raw
`net` package (no HTTP framework). Built for a network architecture
course.

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

The server keeps the TCP connection open across requests instead of
closing after each one. That sounds simple until you remember what TCP
actually gives you.

## Why you can't just read-once-and-respond

TCP only guarantees a stream of bytes — it doesn't know or care where
your HTTP messages start or stop. That means a single read off the
socket might hand you half a request, or a full request plus a few
bytes of the next one that arrived right behind it. So the server
can't treat "one read" as "one request." It has to hold a per-connection
buffer and pull bytes out of it request by request.

To figure out where a request actually ends, the server scans for the
blank line (`\r\n\r\n`) that separates headers from whatever comes
next. Once it hits that, it checks `Content-Length` — nothing in this
assignment's routes actually needs a body, so it's normally zero — and
if it's non-zero, reads exactly that many more bytes before moving on.
This happens even for requests that get rejected, like the `POST /add`
case: if the server skipped the body there, those leftover bytes would
land at the front of the next request and corrupt its parsing. Anything
that arrives after the current request's boundary is simply left alone
in the buffer for the next loop iteration to pick up.

Every response the server sends back also carries an accurate
`Content-Length` of its own, which is the other half of this — it's
how the client knows exactly where one response ends and can start
looking for the next, all on the same socket.

The connection stays open by default, the way HTTP/1.1 is supposed to
work, and only closes when the client disconnects or when something
comes in that breaks the framing badly enough that there's no honest
way to know where the next request would even begin. Each accepted
connection runs in its own goroutine, so several clients can be
connected at once — it's only within a single connection that requests
get handled strictly one after another, in the order they arrive.

## Running it

```bash
go build -o calc-server main.go
./calc-server
# listens on :8080
```

## Testing manually

```bash
single:
curl "http://localhost:8080/add?a=2&b=3"

(or)

all:
chmod +x test_server.sh
./test_server.sh
```

`test_server.sh` opens one raw socket and fires all nine requests down
it back-to-back, checking status codes and bodies, then confirms the
connection is still open at the end.
