# Introduction

## `run.sh`
For local use.
```shell
run.sh <duration> <experiment_name> <builder_count> <validator_count> <non-validator_count>
```
Example:
```shell
run.sh 30 my_exp_with_10_nodes 1 4 5
```

## `test.bat`
For local use.
```shell
test.bat <duration> <builder_count> <validator_count> <non-validator_count>
```
Example:
```shell
test.bat 30 1 1 2
```

## `experiment_launcher.py`
For grid5k use.
Just set values to the following variables in the script:
```txt
login
nb_node
nb_builder
nb_validator
duration_secs
```
Then run using python:
```shell
python3 experiment_launcher.py
```