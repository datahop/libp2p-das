#!/bin/bash

# Script parameters: 
# $1 = duration
# $2 = experiment_name
# $3 = builder count
# $4 = validator count
# $5 = non validator count

duration=$1
experiment_name=$2
builder_count=$3
validator_count=$4
non_validator_count=$5

result_dir="/home/kpeeroo/result"

#Install go
cd /tmp
wget "https://go.dev/dl/go1.20.4.linux-amd64.tar.gz"
sudo-g5k tar -C /usr/local -xzf go1.20.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

#Build and run experiment
git clone https://github.com/Blitz3r123/libp2p-das-datahop.git
cd libp2p-das-datahop

# Run builders
for ((i=1; i<=$builder_count; i++))
    echo "Running builder $i"
    do
        go run . -debug -seed 1234 -port 61960 -nodeType builder -duration $duration &
    done

# Run validators
for ((i=1; i<=$validator_count; i++))
    echo "Running validator $i"
    do
        go run . -debug -duration $duration -nodeType validator -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
    done

# Run non validators
for ((i=1; i<=$non_validator_count; i++))
    echo "Running non validator $i"
    do
        go run . -debug -duration $duration -nodeType nonvalidator -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
    done

if [!-d $result_dir]; then
    mkdir -p $result_dir;
fi;
mkdir $result_dir/$experiment_name_$(date +%d-%m-%y-%H-%M)
cp *.csv $result_dir/$experiment_name_$(date +%d-%m-%y-%H-%M)
rm -rf *.csv
cd ..
rm -rf libp2p-das-datahop