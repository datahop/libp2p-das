import json
import re
from datetime import datetime
from pprint import pprint
from icecream import ic
import os
import pandas as pd
import matplotlib.pyplot as plt
from rich.console import Console
from rich.progress import track

console = Console()
test_data = []
errors = []
non_builder_counts = [1, 2, 3, 4, 5]
parcel_sizes = [256, 512]

def parse_stats(file_path):
    """
    Parses a file containing system stats and returns the CPU and memory usage over time.

    Parameters:
    file_path (str): The path to the stats file.

    Returns:
    tuple: Two lists, one for CPU usage and one for memory usage.
    """
    cpu_usage = []
    memory_usage = []
    network_incoming_usage = []
    network_outgoing_usage = []
    with open(file_path, 'r') as file:
        for line in file:
            # Check for CPU usage
            if 'CPU usage' in line:
                # Extract the total CPU usage (user + sys)
                cpu_numbers = re.findall(r'\d+\.\d+', line)
                if len(cpu_numbers) >= 2:
                    total_cpu = sum(float(num) for num in cpu_numbers[:2])  # Sum of user and sys
                    cpu_usage.append(total_cpu)

            # Check for memory usage
            if 'PhysMem' in line:
                # Extract the used memory
                mem_match = re.search(r'(\d+G) used', line)
                if mem_match:
                    used_memory = mem_match.group(1)
                    memory_usage.append(used_memory)

            # Check for network usage
            if "Networks" in line:
                pattern = re.compile(r'Networks: packets: (\d+)/([\d.]+[G|M]) in, (\d+)/([\d.]+[G|M]) out')
                match = pattern.search(line)

                if match:
                    # Extracting incoming and outgoing network usage and appending to the respective lists
                    network_incoming_usage.append((match.group(1), match.group(2)))  # (packets, data) tuple
                    network_outgoing_usage.append((match.group(3), match.group(4)))  # (packets, data) tuple

    return cpu_usage, memory_usage, network_incoming_usage, network_outgoing_usage

def get_sampling_times(files):
    if len(files) == 0:
        pprint(f"No files found in {files}")
        return None
    
    latency_files = [f for f in files if "latency" in f]

    if len(latency_files) == 0:
        pprint(f"No latency files found in {files}")
        return None
    
    sampling_times = []

    for latency_file in latency_files:
        df = pd.read_csv(latency_file)
        sampling_col_names = [c for c in df.columns if "total sampling" in c.lower()]

        sampling_times.append(df[sampling_col_names[0]].dropna().tolist())

    return sampling_times    

def get_seeding_times(files):
    
    latency_files = [f for f in files if "latency" in f]
    
    if len(latency_files) == 0:
        pprint(f"No latency files found in {files}")
        return None

    latency_file = latency_files[0]
    df = pd.read_csv(latency_file)

    seeding_col_name = [c for c in df.columns if "seeding" in c.lower()][0]
    seeding_times = df[seeding_col_name]
    seeding_times.dropna(inplace=True)

    return seeding_times
    
with console.status("[bold green] Deleting old data..."):
    if os.path.exists("local_test_data.json"):
        os.remove("local_test_data.json")

for non_builder_count in track(non_builder_counts, description="Processing tests..."):
    for parcel_size in parcel_sizes:
        
        test_name = f"1b{non_builder_count}v{non_builder_count}r{parcel_size}p"
        non_builder_count = non_builder_count
        
        if not os.path.exists(f"{test_name}"):
            errors.append(f"Test {test_name} does not exist")
            continue

        files = os.listdir(f"{test_name}")
        files = [f"{test_name}/{f}" for f in files]

        stats_files = [f for f in files if f.endswith("stats.txt")]
        if len(stats_files) == 0:
            errors.append(f"No stats files found in {test_name}")
            continue
    
        stats_file = stats_files[0]

        builder_files = [f for f in files if f.endswith("builder.csv")]
        if len(builder_files) == 0:
            errors.append(f"No builder files found in {test_name}")
            continue

        validator_files = [f for f in files if f.endswith("validator.csv") and "nonvalidator" not in f]
        if len(validator_files) == 0:
            errors.append(f"No validator files found in {test_name}")
            continue
        
        regular_files = [f for f in files if f.endswith("nonvalidator.csv")]
        if len(regular_files) == 0:
            errors.append(f"No regular files found in {test_name}")
            continue

        builder_seeding_times = get_seeding_times(builder_files)
        if builder_seeding_times is None:
            errors.append(f"No builder seeding times found in {test_name}")
            continue

        validator_sampling_times = get_sampling_times(validator_files)
        if validator_sampling_times is None or len(validator_sampling_times) == 0:
            errors.append(f"No validator sampling times found in {test_name}")
            continue

        regular_sampling_times = get_sampling_times(regular_files)
        if regular_sampling_times is None:
            errors.append(f"No regular sampling times found in {test_name}")
            continue

        cpu_usage, memory_usage, network_in_usage, network_out_usage = parse_stats(stats_file)

        single_test_data = {
            "test_name": test_name,
            "non_builder_count": non_builder_count,
            "parcel_size": parcel_size,
            "builder_seeding_times_us": builder_seeding_times,
            "validator_sampling_times_us": validator_sampling_times,
            "regular_sampling_times_us": regular_sampling_times,
            "cpu_usage": cpu_usage,
            "memory_usage": memory_usage,
            "network_in_usage": network_in_usage,
            "network_out_usage": network_out_usage
        }

        test_data.append(single_test_data)

with console.status("[bold green]Writing data to local_test_data.json..."):
    with open("local_test_data.json", "w") as file:
        json.dump(test_data, file, default=lambda x: x.tolist(), indent=4)

if len(errors) > 0:
    console.print(f"{len(errors)} Errors:", style="bold red")
    for error in errors:
        console.print(error, style="bold red")