#!/bin/bash

duration=$1
nodeType=$2
parcelSize=$3

if [ -z "$duration" ]; then
    echo "There should be 3 parameters: duration, nodeType, and parcelSize. e.g. run.sh 30 builder 256"
    exit 1
elif [ -z "$nodeType" ]; then
    echo "There should be 3 parameters: duration, nodeType, and parcelSize. e.g. run.sh 30 builder 256"
    exit 1
elif [ -z "$parcelSize" ]; then
    echo "There should be 3 parameters: duration, nodeType, and parcelSize. e.g. run.sh 30 builder 256"
    exit 1
fi

if [ "$nodeType" != "builder" ] && [ "$nodeType" != "validator" ] && [ "$nodeType" != "nonvalidator" ]; then
    echo "Invalid nodeType. Valid options are \"builder\", \"validator\", or \"nonvalidator\"."
    exit 1
fi

if [ "$nodeType" == "builder" ]; then
    go run . -debug -seed 1234 -port 61960 -nodeType builder -duration $duration -parcelSize $parcelSize
    exit 1
else
    go run . -debug -duration $duration -nodeType $nodeType -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 -parcelSize $parcelSize
    exit 1
fi
