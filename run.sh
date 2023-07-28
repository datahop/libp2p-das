#!/bin/bash

# Script parameters: 
# $1=duration
# $2=experiment_name
# $3=builder count
# $4=validator count
# $5=non validator count

exp_duration=$1
experiment_name=$2
builder_count=$3
validator_count=$4
non_validator_count=$5
login=$6

result_dir="/home/${login}/results"
finish_time=$(date +%d-%m-%y-%H-%M)
result_dir="${result_dir}/${experiment_name}_${finish_time}"

if [ ! -d "$result_dir" ]; then
    mkdir -p "$result_dir"
fi

#Install go
echo "Installing go"
cd /tmp
wget "https://go.dev/dl/go1.20.4.linux-amd64.tar.gz"
sudo-g5k tar -C /usr/local -xzf go1.20.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

echo "Installing libp2p-das-datahop"
#Build and run experiment
git clone https://github.com/Blitz3r123/libp2p-das-datahop.git
cd libp2p-das-datahop

echo "Running builders"
# Run builders
for ((i=1; i<=$builder_count; i++))
do
    echo "Running builder $i"
    go run . -debug -seed 1234 -port 61960 -nodeType builder -duration $duration &
    sleep 0.01
done

echo "Running validators"
# Run validators
for ((i=0; i<=$validator_count - 1; i++))
do
    echo "Running validator $i"
    go run . -debug -duration $duration -nodeType validator -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
    sleep 0.01
done

if [ $(($non_validator_count)) -eq 0 ]
then
    go run . -debug -seed 1234 -port 61960 -nodeType validator -duration $duration
else
    go run . -debug -seed 1234 -port 61960 -nodeType validator -duration $duration &
fi

echo "Running non-validators"
# Run non validators
for ((i=0; i<=$non_validator_count - 1; i++))
do
    echo "Running non validator $i"
    go run . -debug -duration $duration -nodeType nonvalidator -peer /ip4/127.0.0.1/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
    sleep 0.01
done

if [ $(($non_validator_count)) -ne 0 ]; then
    go run . -debug -seed 1234 -port 61960 -nodeType nonvalidators -duration $duration
fi;

mkdir "${result_dir}/${experiment_name}_${finish_time}}"
cp *.csv "${result_dir}/${experiment_name}_${finish_time}}"
