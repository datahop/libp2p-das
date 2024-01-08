#!/bin/bash

# Script parameters: 
# $1 = experiment_name
# $2 = builder count
# $3 = validator count
# $4 = non validator count
# $5 = login
# $6 = builder ip
# $7 = parcel size

experiment_name=$1
builder_count=$2
validator_count=$3
non_validator_count=$4
login=$5
builder_ip=$6
parcel_size=$7

result_dir="/home/${login}/results"
finish_time=$(date +%d-%m-%y-%H-%M)
result_dir="${result_dir}/${experiment_name}_${finish_time}"

rm -rf oar*
rm -rf OAR*

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

sudo-g5k systemctl start sysstat
sar -A -o sar_logs 1 $exp_duration >/dev/null 2>&1 &
sleep 1

# Run builders
for ((i=0; i<$builder_count-1; i++))
do
    echo "[BACKGROUND] Running builder $i"
    go run . -seed 1234 -port 61960 -nodeType builder -parcelSize $parcel_size &
    sleep 1
done
if [ $(($builder_count)) -ne 0 ]; then
    if [ $(($non_validator_count)) -eq 0 ] && [ $(($validator_count)) -eq 0 ]; then
        echo "[FOREGROUND] Running builder [0]"

        go run . -seed 1234 -port 61960 -nodeType builder -parcelSize $parcel_size
        sleep 1
    else
        go run . -seed 1234 -port 61960 -nodeType builder -parcelSize $parcel_size &
        sleep 1
    fi;
fi;

# Run validators
for ((i=0; i<$validator_count - 1; i++))
do
    echo "[BACKGROUND] Running validator $i"
    go run . -nodeType validator -parcelSize $parcel_size -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
done

if [ $(($non_validator_count)) -eq 0 ]
then
    if [ $(($validator_count)) -ne 0 ]; then
        echo "[FOREGROUND] Running validator $i"
        go run . -nodeType validator -parcelSize $parcel_size -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73
        sleep 1
    fi;
else
    echo "[BACKGROUND] Running validator $i"
    go run . -nodeType validator -parcelSize $parcel_size -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
fi

# Run non validators
for ((i=0; i<$non_validator_count - 1; i++))
do
    echo "[BACKGROUND] Running non validator $i"
    go run . -nodeType nonvalidator -parcelSize $parcel_size -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73 &
done

if [ $(($non_validator_count)) -ne 0 ]; then
    echo "[FOREGROUND] Running non validator $i"
    go run . -nodeType nonvalidator -parcelSize $parcel_size -peer /ip4/$builder_ip/tcp/61960/p2p/12D3KooWE3AwZFT9zEWDUxhya62hmvEbRxYBWaosn7Kiqw5wsu73
    sleep 1
fi;

cp *.csv "$result_dir"
cp sar_logs "$result_dir"
sleep 1