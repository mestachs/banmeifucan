#!/bin/bash

kill -9 $(ps aux | grep '[p]ython -m http.server 8080' | awk '{print $2}')
kill -9 $(ps aux | grep './reverse_proxy' | grep -v grep | awk '{print $2}')

go build -o reverse_proxy .

BACKEND_URL=http://127.0.0.1:8080 ./reverse_proxy -hit-404-threshold 10 -ban-duration-in-minutes 1 python -m http.server 8080