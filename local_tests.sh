non_builder_counts=(1 2 5)
# non_builder_counts=(1 2)

parcel_sizes=(512 256)
# parcel_sizes=(512 256 128 64)

total_test_count=$(( ${#non_builder_counts[@]} * ${#parcel_sizes[@]} ))
current_test_count=0

rm -rf *.csv

for i in "${non_builder_counts[@]}"
do
    for j in "${parcel_sizes[@]}"
    do
        current_test_count=$((current_test_count+1))
        echo ""
        echo "[${current_test_count}/${total_test_count}] Running 1b${i}v${i}r${j}p..."
        echo ""
        mkdir "1b${i}v${i}r${j}p"
        gtimeout 5m ./test.sh 1 ${i} ${i} ${j}
        if [ $? -eq 124 ]; then
            echo "Test timed out. Skipping to next test."
            continue
        fi
        mv *.csv "1b${i}v${i}r${j}p"
    done
done