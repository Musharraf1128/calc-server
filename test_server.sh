#!/usr/bin/env bash
#
# Test harness for calc-server.
#
# Opens ONE TCP connection (via bash's /dev/tcp) and fires all 9
# requests from the assignment spec down it sequentially, checking
# status codes and response bodies, then confirms the connection is
# still alive at the end.
#
# Usage:
#   ./calc-server &        # start the server first, listens on :8080
#   ./test_server.sh

set -u

HOST="localhost"
PORT=8080

# Open one persistent TCP connection on file descriptor 3. Every
# request below is written to this same fd - nothing reconnects.
exec 3<>"/dev/tcp/${HOST}/${PORT}"

pass_count=0
fail_count=0

# Reads one full HTTP response off fd 3: the status line, headers up
# to the blank line, then exactly Content-Length body bytes. Nothing
# past that is touched, so the next request's bytes are untouched.
read_response() {
    local status_line=""
    local content_length=0
    local line

    IFS= read -r line <&3
    status_line="${line%$'\r'}"

    while IFS= read -r line <&3; do
        line="${line%$'\r'}"
        [ -z "$line" ] && break
        if [[ "$line" =~ ^Content-Length:\ ([0-9]+)$ ]]; then
            content_length="${BASH_REMATCH[1]}"
        fi
    done

    local body=""
    if [ "$content_length" -gt 0 ]; then
        body=$(dd bs=1 count="$content_length" <&3 2>/dev/null)
    fi

    echo "${status_line}|${body}"
}

# Each test: raw request bytes, expected status code, expected body
# (empty string means "don't check the body, just the status").
run_test() {
    local n="$1"
    local raw="$2"
    local expected_status="$3"
    local expected_body="$4"

    printf '%b' "$raw" >&3

    local result
    result="$(read_response)"
    local status_line="${result%%|*}"
    local body="${result#*|}"

    local ok=1
    [[ "$status_line" == *" ${expected_status} "* ]] || ok=0
    if [ -n "$expected_body" ] && [ "$body" != "$expected_body" ]; then
        ok=0
    fi

    if [ "$ok" -eq 1 ]; then
        echo "[$n] PASS  ${status_line}  body=${body@Q}"
        pass_count=$((pass_count + 1))
    else
        echo "[$n] FAIL  ${status_line}  body=${body@Q}  (expected ${expected_status})"
        fail_count=$((fail_count + 1))
    fi
}

run_test 1 'GET /add?a=2&b=3 HTTP/1.1\r\nHost: localhost\r\n\r\n' "200" "5"
run_test 2 'GET /sub?a=10&b=4 HTTP/1.1\r\nHost: localhost\r\n\r\n' "200" "6"
run_test 3 'GET /mul?a=6&b=7 HTTP/1.1\r\nHost: localhost\r\n\r\n' "200" "42"
run_test 4 'GET /div?a=9&b=3 HTTP/1.1\r\nHost: localhost\r\n\r\n' "200" "3"
run_test 5 'GET /div?a=1&b=0 HTTP/1.1\r\nHost: localhost\r\n\r\n' "400" ""
run_test 6 'GET /add?a=x&b=3 HTTP/1.1\r\nHost: localhost\r\n\r\n' "400" ""
run_test 7 'GET /pow?a=2&b=8 HTTP/1.1\r\nHost: localhost\r\n\r\n' "404" ""
run_test 8 'POST /add HTTP/1.1\r\nHost: localhost\r\nContent-Length: 0\r\n\r\n' "405" ""
run_test 9 'GET /add?a=1&b=1 HTTP/1.1\r\n\r\n' "400" ""

# Confirm the socket is still usable - if the server closed it, this
# fd would already be dead and this write would fail.
socket_alive=1
if ! : >&3 2>/dev/null; then
    socket_alive=0
fi

exec 3<&-
exec 3>&-

echo
echo "passed: ${pass_count}, failed: ${fail_count}"
echo "socket still open: $([ "$socket_alive" -eq 1 ] && echo true || echo false)"

[ "$fail_count" -eq 0 ]
