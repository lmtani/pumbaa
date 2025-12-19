# Resource Monitoring Script

Pumbaa provides a simple script to monitor computational resource usage over time. This script is designed to be used as a **monitoring script** in Cromwell, allowing you to estimate the resource usage efficiency for each task.

## Download

[:material-download: Download resource_monitor.sh](../assets/resource_monitor.sh){ .md-button .md-button--primary }

## Features

- **Lightweight**: Uses only native Linux tools (`/proc` filesystem).
- **No dependencies**: Works on most Linux distributions.
- **Complete metrics**: CPU, memory, disk I/O, disk space, and network.
- **TSV format**: Easy to analyze with tools like `awk`, `pandas`, or Excel.

## Collected Metrics

| Column | Description |
|--------|-----------|
| `timestamp` | Date and time of the measurement |
| `cpu_percent` | CPU usage percentage (all cores) |
| `mem_used_mb` | Used memory in MB |
| `mem_total_mb` | Total memory in MB |
| `mem_percent` | Memory usage percentage |
| `disk_total_gb` | Total disk space in GB |
| `disk_used_gb` | Used disk space in GB |
| `disk_avail_gb` | Available disk space in GB |
| `disk_percent` | Disk usage percentage |
| `disk_read_mb` | MB read from disk since last measurement |
| `disk_write_mb` | MB written to disk since last measurement |
| `net_rx_mb` | MB received via network since last measurement |
| `net_tx_mb` | MB transmitted via network since last measurement |

## Usage

### Syntax

```bash
./resource_monitor.sh [interval_seconds] [disk_path]
```

The script outputs metrics to **stdout** in TSV format. Use redirection to save to a file.

### Parameters

| Parameter | Default | Description |
|-----------|--------|-----------|
| `interval_seconds` | 10 | Time between measurements in seconds |
| `disk_path` | `/mnt/disks/cromwell_root/` | Disk path to monitor space for |

### Examples

**Basic usage** (10-second interval, Cromwell's default disk):
```bash
./resource_monitor.sh
```

**Save to a file**:
```bash
./resource_monitor.sh > resource_metrics.tsv
```

**Custom interval** (5 seconds):
```bash
./resource_monitor.sh 5 > resource_metrics.tsv
```

**Monitor a different disk**:
```bash
./resource_monitor.sh 10 /data > resource_metrics.tsv
```

## Cromwell Integration

To use the script as a Cromwell monitoring script, add it to your backend configuration:

```hocon
backend {
  providers {
    Local {
      config {
        # Executes the monitoring script for each task
        submit = """
          chmod +x ${cwd}/resource_monitor.sh
          ${cwd}/resource_monitor.sh 10 > ${cwd}/resource_metrics.tsv &
          MONITOR_PID=$!
          
          # Your original command here
          ${job_shell} ${script}
          EXIT_CODE=$?
          
          # Stop monitoring when the task finishes
          kill $MONITOR_PID 2>/dev/null
          exit $EXIT_CODE
        """
      }
    }
  }
}
```

!!! tip "Tip"
    Copy the script to a location accessible by Cromwell (e.g., alongside your WDL files) or include it as an input in your workflow.

## Analyzing Results

### Using awk

```bash
# Average CPU usage
awk -F'\t' 'NR>1 {sum+=$2; count++} END {print "Average CPU:", sum/count"%"}' resource_metrics.tsv

# Peak memory usage
awk -F'\t' 'NR>1 {if($5>max) max=$5} END {print "Peak Memory:", max"%"}' resource_metrics.tsv
```

### Using Python/Pandas

```python
import pandas as pd

df = pd.read_csv('resource_metrics.tsv', sep='\t')
print(df.describe())

# Plot CPU usage over time
df.plot(x='timestamp', y='cpu_percent', title='CPU Usage Over Time')
```

## Estimating Efficiency

With the collected data, you can calculate resource usage efficiency:

$$
\text{Efficiency} = \frac{\text{Resource Used}}{\text{Resource Allocated}} \times 100\%
$$

For example, if you allocated 16GB of RAM to a task that used a maximum of 8GB, the memory efficiency was 50%.

!!! info "About efficiency"
    Low efficiency indicates that you could potentially reduce the resources allocated to the task, saving costs in cloud environments.
