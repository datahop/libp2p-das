#!/bin/bash

nodeType=$1
parcelSize=$2

if [ -z "$nodeType" ] || [ -z "$parcelSize" ]; then
    echo "There should be 2 parameters: nodeType, and parcelSize. e.g. run_node.sh builder 256"
    exit 1
fi

if [ "$nodeType" != "builder" ] && [ "$nodeType" != "validator" ] && [ "$nodeType" != "nonvalidator" ]; then
    echo "Invalid nodeType. Valid options are 'builder', 'validator', or 'nonvalidator'."
    exit 1
fi

if [ "$nodeType" == "builder" ]; then
    go run . -seed 1234 -port 61960 -nodeType builder -parcelSize $parcelSize
    exit 1
else
    go run . -nodeType $nodeType -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 -parcelSize $parcelSize
    exit 1
fi
