#!/bin/bash

rm -rf ./*_builder.csv
rm -rf ./*_validator.csv
rm -rf ./*_nonvalidator.csv

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

# echo "Starting $builderCount builder(s), $validatorCount validator(s), and $nonValidatorCount non-validator(s) with parcel size $parcelSize..."

# Create an array to store background process IDs
declare -a bg_pids

trap 'echo "Stopping all processes"; pkill -P $$; exit 1' SIGINT

for ((i=1; i<=$builderCount; i++)); do
    ./run_node.sh builder $parcelSize &
    bg_pids+=($!)  # Store the background process ID in the array
done
# echo "Buiders started."

for ((i=1; i<=$validatorCount; i++)); do
    ./run_node.sh validator $parcelSize &
    bg_pids+=($!)
done
# echo "Validators started."

for ((i=1; i<=$nonValidatorCount; i++)); do
    ./run_node.sh nonvalidator $parcelSize &
    bg_pids+=($!)
done
# echo "Non-validators started."

# Wait for all background processes to finish
for pid in "${bg_pids[@]}"; do
    wait $pid
done

echo "All processes have finished."