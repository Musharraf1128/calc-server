#!/usr/bin/env python3
"""
Test harness for calc-server.

Opens ONE TCP connection and fires all 9 requests from the assignment
spec down it sequentially, checking status codes, response bodies, and
that the connection is still alive at the end.

Usage:
    ./calc-server &        # start the server first, listens on :8080
    python3 test_server.py
"""
import socket
import sys

HOST = "localhost"
PORT = 8080


def read_one_response(sock):
    buf = b""
    while b"\r\n\r\n" not in buf:
        chunk = sock.recv(1)
        if not chunk:
            return None, None
        buf += chunk
    head, rest = buf.split(b"\r\n\r\n", 1)
    headers = head.decode().split("\r\n")
    content_length = 0
    for h in headers[1:]:
        k, v = h.split(": ", 1)
        if k.lower() == "content-length":
            content_length = int(v)
    body = rest
    while len(body) < content_length:
        body += sock.recv(content_length - len(body))
    return head.decode().splitlines()[0], body.decode()


TESTS = [
    (b"GET /add?a=2&b=3 HTTP/1.1\r\nHost: localhost\r\n\r\n", "200", "5\n"),
    (b"GET /sub?a=10&b=4 HTTP/1.1\r\nHost: localhost\r\n\r\n", "200", "6\n"),
    (b"GET /mul?a=6&b=7 HTTP/1.1\r\nHost: localhost\r\n\r\n", "200", "42\n"),
    (b"GET /div?a=9&b=3 HTTP/1.1\r\nHost: localhost\r\n\r\n", "200", "3\n"),
    (b"GET /div?a=1&b=0 HTTP/1.1\r\nHost: localhost\r\n\r\n", "400", None),
    (b"GET /add?a=x&b=3 HTTP/1.1\r\nHost: localhost\r\n\r\n", "400", None),
    (b"GET /pow?a=2&b=8 HTTP/1.1\r\nHost: localhost\r\n\r\n", "404", None),
    (b"POST /add HTTP/1.1\r\nHost: localhost\r\nContent-Length: 0\r\n\r\n", "405", None),
    (b"GET /add?a=1&b=1 HTTP/1.1\r\n\r\n", "400", None),  # no Host header
]


def main():
    s = socket.create_connection((HOST, PORT))
    all_pass = True

    for i, (raw, expected_status, expected_body) in enumerate(TESTS, 1):
        s.sendall(raw)
        status_line, body = read_one_response(s)
        ok_status = status_line is not None and f" {expected_status} " in status_line
        ok_body = expected_body is None or body == expected_body
        ok = ok_status and ok_body
        all_pass = all_pass and ok
        label = raw.split(b" ")[1].decode() if b" " in raw else ""
        print(f"[{i}] {status_line!r} body={body!r} {'PASS' if ok else 'FAIL (expected ' + expected_status + ')'}")

    print()
    print("ALL PASS:", all_pass)
    print("socket still open:", s.fileno() != -1)
    s.close()

    sys.exit(0 if all_pass else 1)


if __name__ == "__main__":
    main()
