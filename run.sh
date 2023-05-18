#!/bin/bash

if [[ $1 == "bootstrap" ]]; then
    go run . -seed 1234 &
else
    go run . -peer /ip4/10.210.118.52/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
fi