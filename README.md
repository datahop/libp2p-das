# How to run
## Normal
```shell
./run.sh <is_bootstrap> <execution_duration_in_secs>
```

Example:
Running 3 peers with 1 being the bootstrap node for 20 seconds.
```shell
./run.sh bootstrap 20
./run.sh 20
./run.sh 20
```

## Testing
```shell
./test.sh <node_count> <debug_mode>
```

Example: 
Running 3 peers.
```shell
./test.sh 3
```

# Logging
What do I log?
- # of put message
- # of get messages
    - # of successful gets
    - # of failed gets
- latencies per message