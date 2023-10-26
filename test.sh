#!/bin/bash

rm -f *.csv

duration=$1
builderCount=$2
validatorCount=$3
nonValidatorCount=$4
parcelSize=$5

if [ -z "$duration" ]; then
    echo "There should be 5 parameters: duration, builderCount, validatorCount, nonValidatorCount, and parcelSize. e.g. test.sh 30 1 2 1 512"
    exit 1
elif [ -z "$builderCount" ]; then
    echo "There should be 5 parameters: duration, builderCount, validatorCount, nonValidatorCount, and parcelSize. e.g. test.sh 30 1 2 1 512"
    exit 1
elif [ -z "$validatorCount" ]; then
    echo "There should be 5 parameters: duration, builderCount, validatorCount, nonValidatorCount, and parcelSize. e.g. test.sh 30 1 2 1 512"
    exit 1
elif [ -z "$nonValidatorCount" ]; then
    echo "There should be 5 parameters: duration, builderCount, validatorCount, nonValidatorCount, and parcelSize. e.g. test.sh 30 1 2 1 512"
    exit 1
elif [ -z "$parcelSize" ]; then
    echo "There should be 5 parameters: duration, builderCount, validatorCount, nonValidatorCount, and parcelSize. e.g. test.sh 30 1 2 1 512"
    exit 1
fi

builderCount=$((builderCount))
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
    ./run_node.sh $duration builder $parcelSize &
done

for ((i=1; i<=$validatorCount; i++)); do
    ./run_node.sh $duration validator $parcelSize &
done

for ((i=1; i<=$nonValidatorCount; i++)); do
    ./run_node.sh $duration nonvalidator $parcelSize &
done
