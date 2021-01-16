#!/bin/bash
trap "rm server;kill 0" EXIT

go build -o server
./server -port=8051 &
./server -port=8052 &
./server -port=8053 -api=1 &

sleep 2
echo ">>> start test"
curl "http://localhost:9994/api?key=Tom" &
curl "http://localhost:9994/api?key=Tom" &
curl "http://localhost:9994/api?key=Tom" &
curl "http://localhost:9994/api?key=Lily" &
curl "http://localhost:9994/api?key=Lily" &
curl "http://localhost:9994/api?key=Lily" &
curl "http://localhost:9994/api?key=Pity" &
curl "http://localhost:9994/api?key=Pity" &
curl "http://localhost:9994/api?key=Pity" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
curl "http://localhost:9994/api?key=Sam" &
wait