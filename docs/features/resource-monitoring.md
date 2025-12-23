# Resource Monitoring Script

Monitor computational resource usage for Cromwell tasks to estimate efficiency.

[:material-download: Download resource_monitor.sh](../assets/resource_monitor.sh){ .md-button .md-button--primary }

---

## :sparkles: Features

<div class="grid cards" markdown>

-   :material-feather: **Lightweight**
    
    Uses only native Linux tools (`/proc` filesystem)

-   :material-package-variant-closed: **No Dependencies**
    
    Works on most Linux distributions

-   :material-chart-line: **Complete Metrics**
    
    CPU, memory, disk I/O, disk space, and network

-   :material-file-delimited: **TSV Format**
    
    Easy to analyze with `awk`, `pandas`, or Excel

</div>

---

## :bar_chart: Collected Metrics

| Column | Description |
|--------|-------------|
| `timestamp` | Date and time |
| `cpu_percent` | CPU usage (all cores) |
| `mem_used_mb` | Used memory (MB) |
| `mem_total_mb` | Total memory (MB) |
| `mem_percent` | Memory usage % |
| `disk_total_gb` | Total disk (GB) |
| `disk_used_gb` | Used disk (GB) |
| `disk_avail_gb` | Available disk (GB) |
| `disk_percent` | Disk usage % |
| `disk_read_mb` | MB read since last measurement |
| `disk_write_mb` | MB written since last measurement |
| `net_rx_mb` | Network MB received |
| `net_tx_mb` | Network MB transmitted |

---

## :rocket: Usage

### Syntax

```bash
./resource_monitor.sh [interval_seconds] [disk_path]
```

### Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `interval_seconds` | `10` | Time between measurements |
| `disk_path` | `/mnt/disks/cromwell_root/` | Disk path to monitor |

### Examples

=== "Basic"

    ```bash
    ./resource_monitor.sh
    ```

=== "Save to File"

    ```bash
    ./resource_monitor.sh > resource_metrics.tsv
    ```

=== "Custom Interval"

    ```bash
    ./resource_monitor.sh 5 > resource_metrics.tsv
    ```

---

## :gear: Cromwell Integration

To use this script as a Cromwell monitoring script, configure it in your backend options.

!!! info "Cromwell Documentation"
    For detailed configuration instructions, see the official [Cromwell Workflow Options documentation](https://cromwell.readthedocs.io/en/stable/wf_options/Google/).

