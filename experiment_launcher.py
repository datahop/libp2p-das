import logging
from icecream import ic
import enoslib as en
import os
import datetime
import subprocess
import time
import sys
from rich.console import Console
from rich.progress import track

console = Console()

#Upload launch script to site frontend
def execute_ssh_command(ssh_command, login, site):
    try:
        # Execute the SSH command
        result = subprocess.run(ssh_command, shell=True, capture_output=True, text=True)
        # Check if the command was successful
        if result.returncode == 0:
            # Print the output
            print(result.stdout)
        else:
            # Print the error message
            print(result.stderr)

    except subprocess.CalledProcessError as e:
        print(f"Error occurred while executing SSH command: {e}")

#Get timestamp after end of experiment
def add_time(original_time, hours=0, minutes=0, seconds=0):
    time_delta = datetime.timedelta(hours=hours, minutes=minutes, seconds=seconds)
    new_time = original_time + time_delta
    return new_time

def convert_seconds_to_time(seconds):
    hours, remainder = divmod(seconds, 3600)
    minutes, seconds = divmod(remainder, 60)
    return hours, minutes, seconds

def seconds_to_hh_mm_ss(seconds):
    hours = seconds // 3600
    minutes = (seconds % 3600) // 60
    seconds = seconds % 60
    return f"{hours:02d}:{minutes:02d}:{seconds:02d}"

#Experiment node partition between Grid5000 machine
def node_partition(nb_cluster_machine, nb_builder, nb_validator, nb_regular):
    partition = [[0, 0, 0] for i in range(nb_cluster_machine)]

    index = 0
    while nb_builder > 0 or nb_validator > 0 or nb_regular > 0:
        if index == len(partition):
            index  = 0
        if nb_builder > 0:
            partition[index][0] += 1
            nb_builder -= 1
        elif nb_validator > 0:
            partition[index][1] += 1
            nb_validator -= 1            
        elif nb_regular > 0:
            partition[index][2] += 1
            nb_regular -= 1      
        index += 1
    return partition

def main(output_dir):
    #========== Parameters ==========
    #Grid5000 parameters
    login = "kpeeroo" #Grid5000 login
    site = "nancy" #Grid5000 Site See: https://www.grid5000.fr/w/Status and https://www.grid5000.fr/w/Hardware
    cluster = "gros" #Gride5000 Cluster name See: https://www.grid5000.fr/w/Status and https://www.grid5000.fr/w/Hardware
    job_name = "PANDAS_libp2p"

    #Node launch script path
    dir_path = os.path.dirname(os.path.realpath(__file__)) #Get current directory path
    launch_script = dir_path +"/" + "run.sh"

    #Experiment parameters
    parcel_size = 512
    nb_cluster_machine = 3         #Number of machine booked on the cluster
    nb_experiment_node = 3        #Number of nodes running for the experiment
    nb_builder = 1
    nb_validator = 1
    nb_regular = nb_experiment_node - nb_builder - nb_validator
    experiment_name = f"PANDAS_libp2p_{nb_builder}b_{nb_validator}v_{nb_regular}r_{parcel_size}p_"
    current_datetime = datetime.datetime.now()
    experiment_name += current_datetime.strftime("%Y-%m-%d-%H:%M:%S")
    walltime_secs = 300 #Experiment walltime in seconds
    
    #Network parameters 
    delay = "10%"
    rate = "1gbit"
    loss = "0%"
    symmetric=True

    #========== Experiment nodes partition on cluster machines ==========
    partition = node_partition(nb_cluster_machine, nb_builder, nb_validator, nb_regular)

    #========== Create and validate Grid5000 and network emulation configurations ==========
    #Log to Grid5000 and check connection
    en.init_logging(level=logging.INFO)
    en.check()
    network = en.G5kNetworkConf(type="prod", roles=["experiment_network"], site=site)
    
    job_walltime = seconds_to_hh_mm_ss(walltime_secs)

    conf = (
        en.G5kConf.from_settings(job_name=job_name, walltime=job_walltime)
        .add_network_conf(network)
        .add_machine(roles=["experiment"], cluster=cluster, nodes=nb_cluster_machine, primary_network=network) # Add experiment nodes
        .finalize()
    )

    #Validate Grid5000 configuration
    start = datetime.datetime.now() #Timestamp grid5000 job start
    provider = en.G5k(conf)
    roles, networks = provider.init()
    roles = en.sync_info(roles, networks)

    # #========== Grid5000 network emulation configuration ==========
    # #network parameters
    # netem = en.NetemHTB()
    # (
    #     netem.add_constraints(
    #         src=roles["experiment"],
    #         dest=roles["experiment"],
    #         delay=delay,
    #         rate=rate,
    #         loss=loss,
    #         symmetric=symmetric,
    #     )
    # )
    
    # #Deploy network emulation
    # netem.deploy()
    # netem.validate()

    #========== Deploy Experiment ==========
    #Send launch script to Grid5000 site frontend
    ssh_command = f'scp {launch_script} {login}@access.grid5000.fr:{site}'
    execute_ssh_command(ssh_command, login, site)
    i = 0

    results = en.run_command("ip -o -4 addr show scope global | awk '!/^[0-9]+: lo:/ {print $4}' | cut -d '/' -f 1", roles=roles["experiment"][0])
    ip = results[0].payload["stdout"]

    for x in roles["experiment"]:
        with en.actions(roles=x, on_error_continue=True, background=True) as p:
            if x == roles["experiment"][0]:
                builder, validator, regular = partition[i]
                p.shell(f"/home/{login}/run.sh {experiment_name} {builder} {validator} {regular} {login} 127.0.0.1 {parcel_size}")
                i += 1
            else:
                builder, validator, regular = partition[i]
                p.shell(f"/home/{login}/run.sh {experiment_name} {builder} {validator} {regular} {login} {ip} {parcel_size}")
                i += 1
    
    start = datetime.datetime.now() #Timestamp grid5000 job start
    start_formatted = start.strftime("%H:%M:%S")
    
    console.print("Start: ", start_formatted, style="bold green")
    console.print("Expected End: ", add_time(start, seconds=walltime_secs).strftime("%H:%M:%S"), style="bold green")

    for i in track(range(walltime_secs + 30), description="Waiting for walltime..."):
        time.sleep(1)

    """
    if output_dir != None:
        
        1. Get all folders in remote results folder
        2. Get all folders in local folder
        3. Find the ones that are remote and not in local folder
        4. Download them
        5. Remove them from remote folder
        

        results_dir = f"/results"

        # Get all folders in remote results folder
        remote_folders = f"ssh {login}@access.grid5000.fr ls {site}{results_dir}"
        remote_folders = subprocess.run(remote_folders, shell=True, stdout=subprocess.PIPE).stdout.decode("utf-8").split("\n")
        remote_folders = [folder for folder in remote_folders if folder != ""]
        
        # Get all folders in local folder
        local_folders = [f for f in os.listdir(output_dir) if os.path.isdir(os.path.join(output_dir, f))]
        
        # Find the ones that are remote and not in local folder
        folders_to_download = [folder for folder in remote_folders if folder not in local_folders]
        
        # Download them
        for folder in folders_to_download:
            remote_path = os.path.join(results_dir, folder)
            local_path = os.path.join(output_dir, folder)
            subprocess.run(f"scp -rC {login}@access.grid5000.fr:{site}{remote_path} {local_path}")
        
        # Remove them from remote folder
        
        for folder in folders_to_download:
            remote_path = os.path.join(results_dir, folder)
            subprocess.run(["ssh", f"{login}@access.grid5000.fr", f"rm -rf {site}{remote_path}"])
        """
    
    #Release all Grid'5000 resources
    # netem.destroy()
    provider.destroy()

if __name__ == "__main__":
    # Check if argument is sent in and is a valid dir path
    if len(sys.argv) > 1:
        dir_path = sys.argv[1]
        if not os.path.isdir(dir_path):
            console.print(f"{dir_path} is an invalid directory path", style="bold red")
            main(None)
        else:
            main(dir_path)
    else:
        main(None)
