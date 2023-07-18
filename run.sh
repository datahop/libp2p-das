#!/bin/bash

# if [[ $1 == "bootstrap" ]]; then
#     go run . -seed 1234 -port 61960 -duration $2 &
# else
#     go run . -peer /ip4/0.0.0.0/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 -duration $1 &
# fi

# # Wait for the code to finish and exit
# wait $!
 
duration=$1
nodeType=$2

if [ -z "$duration" ] || [ -z "$nodeType" ]; then
    echo "There should be 2 parameters: duration and nodeType. e.g. run.sh 30 builder"
    exit 1
fi

if [ "$nodeType" != "builder" ] && [ "$nodeType" != "validator" ] && [ "$nodeType" != "nonvalidator" ]; then
    echo "Invalid nodeType. Valid options are \"builder\", \"validator\", or \"nonvalidator\"."
    exit 1
fi

if [ "$nodeType" == "builder" ]; then
    go run . -debug -seed 1234 -port 61960 -nodeType builder -duration "$duration" & 
else
    go run . -debug -duration "$duration" -nodeType "$nodeType" -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
fi
