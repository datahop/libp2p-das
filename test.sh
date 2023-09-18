#!/bin/bash

# Remove existing CSV files
rm -f *.csv

# Set script parameters
duration="$1"
builderCount="$2"
validatorCount="$3"
nonValidatorCount="$4"
parcelSize="$5"

# Check if parameters are provided
if [ -z "$duration" ] || [ -z "$builderCount" ] || [ -z "$validatorCount" ] || [ -z "$nonValidatorCount" ] || [ -z "$parcelSize" ]; then
    echo "There should be 5 parameters: duration, builderCount, validatorCount, nonValidatorCount, and parcelSize. e.g. ./test.sh 30 1 2 1 512"
    exit 1
fi

# Convert builderCount and nonValidatorCount to integers
builderCount=$((builderCount))

# Check if builderCount is greater than 0
if [ "$builderCount" -le 0 ]; then
    echo "builderCount should be greater than 0."
    exit 1
fi

# Calculate nonBuilderCount
nonBuilderCount=$((validatorCount + nonValidatorCount))

# Check if the sum of validatorCount and nonValidatorCount is greater than 0
if [ "$nonBuilderCount" -le 0 ]; then
    echo "The sum of validatorCount and nonValidatorCount should be greater than 0."
    exit 1
fi

# Start processes in the background
for ((i=1; i<=$nonBuilderCount; i++)); do
    ./run_local.sh "$duration" "nonbuilder" "$parcelSize" &
done