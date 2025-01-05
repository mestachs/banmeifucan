#!/bin/bash

kill -9 $(ps aux | grep '[p]ython -m http.server 8080' | awk '{print $2}')
kill -9 $(ps aux | grep './reverse_proxy' | grep -v grep | awk '{print $2}')

set -e  # Exit the script if any command fails

go build -o reverse_proxy .

go test -coverprofile=coverage.out
go tool cover -func=coverage.out

BACKEND_URL=http://localhost:8080 ./reverse_proxy -modify-host -disable-ban -hit-404-threshold 10 -ban-duration-in-minutes 1 python ./script/jitter.py