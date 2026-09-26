package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// writeResponse sends a full HTTP/1.1 response, correct Content-Length + connection stays open explicitly.
// also conn close is decided by caller's loop close conn ourselves here - the caller's loop decides that.
func writeResponse(conn net.Conn, status int, statusText, body string) error {
	resp := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Length: %d\r\nConnection: keep-alive\r\n\r\n%s",
		status, statusText, len(body), body,
	)
	_, err := conn.Write([]byte(resp))
	return err
}

// parseNumber rejects anything that isn't a clean integer - "x", "",
func parseNumber(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		requestLine, err := reader.ReadString('\n')
		if err != nil {
			return // client close - normal end of seesion
		}
		requestLine = strings.TrimRight(requestLine, "\r\n")

		parts := strings.Split(requestLine, " ")
		if len(parts) != 3 {
			// malformed request line - can't safely keep parsing this
			// connection, so bail rather than guess.
			writeResponse(conn, 400, "Bad Request", "malformed request line\n")
			return
		}
		method, target := parts[0], parts[1]

		headers := make(map[string]string)
		contentLength := 0
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break // end of this request's headers
			}
			if i := strings.Index(line, ": "); i != -1 {
				key := line[:i]
				val := line[i+2:]
				headers[strings.ToLower(key)] = val
				if strings.ToLower(key) == "content-length" {
					if n, ok := parseNumber(val); ok {
						contentLength = n
					}
				}
			}
		}

		// If there's a body (e.g. a POST), we MUST consume exactly
		// Content-Length bytes even though we're about to reject the
		// request - otherwise those leftover body bytes bleed into our
		// parse of the *next* request on this same socket.
		if contentLength > 0 {
			if _, err := io.CopyN(io.Discard, reader, int64(contentLength)); err != nil {
				return
			}
		}

		// Missing Host header -> 400.
		if _, ok := headers["host"]; !ok {
			if err := writeResponse(conn, 400, "Bad Request", "missing Host header\n"); err != nil {
				return
			}
			continue
		}

		// Wrong method -> 405.
		if method != "GET" {
			if err := writeResponse(conn, 405, "Method Not Allowed", "method not allowed\n"); err != nil {
				return
			}
			continue
		}

		u, err := url.Parse(target)
		if err != nil {
			if err := writeResponse(conn, 400, "Bad Request", "malformed target\n"); err != nil {
				return
			}
			continue
		}

		// Unknown route -> 404.
		op := u.Path
		if op != "/add" && op != "/sub" && op != "/mul" && op != "/div" {
			if err := writeResponse(conn, 404, "Not Found", "not found\n"); err != nil {
				return
			}
			continue
		}

		q := u.Query()
		aStr, bStr := q.Get("a"), q.Get("b")
		a, aOK := parseNumber(aStr)
		b, bOK := parseNumber(bStr)
		if !aOK || !bOK {
			if err := writeResponse(conn, 400, "Bad Request", "non-numeric input\n"); err != nil {
				return
			}
			continue
		}

		var result int
		switch op {
		case "/add":
			result = a + b
		case "/sub":
			result = a - b
		case "/mul":
			result = a * b
		case "/div":
			if b == 0 {
				if err := writeResponse(conn, 400, "Bad Request", "division by zero\n"); err != nil {
					return
				}
				continue
			}
			result = a / b
		}

		body := fmt.Sprintf("%d\n", result)
		if err := writeResponse(conn, 200, "OK", body); err != nil {
			return
		}
		// loop back for the next request on this same connection.
	}
}

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("listening on :8080")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handleConn(conn)
	}
}
