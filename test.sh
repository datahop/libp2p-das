#!/bin/bash

if [ $# -ge 1 ]; then
    # Check if the second parameter is 'debug'
    if [ $# -eq 2 ] && [ $2 == "debug" ]; then
        echo "Starting libp2p-das with $1 peers in debug mode."
    else
        echo "Starting libp2p-das with $1 peers."
    fi
else
    echo "Please enter at least 1 parameter. The first parameter is a number and the second parameter is optional and can be 'debug'."
fi

go run . -debug -seed 1234 -port 61960 -duration 15 &
sleep 2

for ((i=1;i<=$1-1;i++));
do
    if [ $# -eq 2 ] && [ $2 == "debug" ]; then
        go run . -debug -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 -duration 15&
        sleep 1
    else 
        go run . -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 -duration 15 &
        sleep 1
    fi
done

wait $!