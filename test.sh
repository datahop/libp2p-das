#!/bin/bash

if [ -z "$1" ] || [ -z "$2" ] || [ -z "$3" ] || [ -z "$4" ]; then
    echo "There should be 4 parameters: builderCount, validatorCount, nonValidatorCount, and parcelSize. e.g. test.sh 1 2 1 512"
    exit 1
fi

builderCount=$1
validatorCount=$2
nonValidatorCount=$3
parcelSize=$4

if [ $builderCount -le 0 ]; then
    echo "builderCount should be greater than 0."
    exit 1
fi

nonBuilderCount=$((validatorCount + nonValidatorCount))

if [ $nonBuilderCount -le 0 ]; then
    echo "The sum of validatorCount and nonValidatorCount should be greater than 0."
    exit 1
fi

for ((i=1; i<=$builderCount; i++)); do
    ./run_node.sh builder $parcelSize &
done

for ((i=1; i<=$validatorCount; i++)); do
    ./run_node.sh validator $parcelSize &
done

for ((i=1; i<=$nonValidatorCount; i++)); do
    ./run_node.sh nonvalidator $parcelSize &
done
