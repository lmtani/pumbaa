#!/bin/bash
#
# Resource Monitor Script for Cromwell Tasks
# Collects CPU, memory, disk I/O, and network metrics every N seconds
# Outputs TSV format to stdout
#
# Usage: ./resource_monitor.sh [interval_seconds] [disk_path]
#   interval_seconds: Time between measurements (default: 10)
#   disk_path: Path to monitor disk space (default: /mnt/disks/cromwell_root/)
#

INTERVAL=${1:-10}
DISK_PATH=${2:-/mnt/disks/cromwell_root/}

# Get number of CPUs for CPU percentage calculation
NUM_CPUS=$(nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo)

# Write header to stdout
echo -e "timestamp\tcpu_percent\tmem_used_mb\tmem_total_mb\tmem_percent\tdisk_total_gb\tdisk_used_gb\tdisk_avail_gb\tdisk_percent\tdisk_read_mb\tdisk_write_mb\tnet_rx_mb\tnet_tx_mb"

# Initialize previous values for delta calculations
prev_cpu_idle=0
prev_cpu_total=0
prev_disk_read=0
prev_disk_write=0
prev_net_rx=0
prev_net_tx=0
first_run=true

while true; do
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    # CPU usage from /proc/stat
    read -r cpu user nice system idle iowait irq softirq steal guest guest_nice < /proc/stat
    cpu_total=$((user + nice + system + idle + iowait + irq + softirq + steal))
    cpu_idle=$idle
    
    if [ "$first_run" = true ]; then
        cpu_percent=0
    else
        cpu_delta=$((cpu_total - prev_cpu_total))
        idle_delta=$((cpu_idle - prev_cpu_idle))
        if [ $cpu_delta -gt 0 ]; then
            cpu_percent=$(awk "BEGIN {printf \"%.2f\", 100 * (1 - $idle_delta / $cpu_delta)}")
        else
            cpu_percent=0
        fi
    fi
    prev_cpu_total=$cpu_total
    prev_cpu_idle=$cpu_idle
    
    # Memory usage from /proc/meminfo
    mem_total=$(awk '/^MemTotal:/ {print $2}' /proc/meminfo)
    mem_available=$(awk '/^MemAvailable:/ {print $2}' /proc/meminfo)
    
    # Fallback if MemAvailable not present (older kernels)
    if [ -z "$mem_available" ]; then
        mem_free=$(awk '/^MemFree:/ {print $2}' /proc/meminfo)
        buffers=$(awk '/^Buffers:/ {print $2}' /proc/meminfo)
        cached=$(awk '/^Cached:/ {print $2}' /proc/meminfo)
        mem_available=$((mem_free + buffers + cached))
    fi
    
    mem_used=$((mem_total - mem_available))
    mem_used_mb=$(awk "BEGIN {printf \"%.2f\", $mem_used / 1024}")
    mem_total_mb=$(awk "BEGIN {printf \"%.2f\", $mem_total / 1024}")
    mem_percent=$(awk "BEGIN {printf \"%.2f\", 100 * $mem_used / $mem_total}")
    
    # Disk I/O from /proc/diskstats (sum all devices)
    disk_read=0
    disk_write=0
    while read -r _ _ dev reads _ _ read_sectors writes _ _ write_sectors _; do
        # Only count physical disks (sd*, nvme*, vd*), not partitions
        if [[ $dev =~ ^(sd[a-z]|nvme[0-9]+n[0-9]+|vd[a-z])$ ]]; then
            disk_read=$((disk_read + read_sectors))
            disk_write=$((disk_write + write_sectors))
        fi
    done < /proc/diskstats
    
    # Convert sectors to MB (sector = 512 bytes)
    if [ "$first_run" = true ]; then
        disk_read_mb=0
        disk_write_mb=0
    else
        disk_read_delta=$((disk_read - prev_disk_read))
        disk_write_delta=$((disk_write - prev_disk_write))
        disk_read_mb=$(awk "BEGIN {printf \"%.2f\", $disk_read_delta * 512 / 1048576}")
        disk_write_mb=$(awk "BEGIN {printf \"%.2f\", $disk_write_delta * 512 / 1048576}")
    fi
    prev_disk_read=$disk_read
    prev_disk_write=$disk_write
    
    # Disk space from df (Cromwell disk path)
    read -r disk_total_kb disk_used_kb disk_avail_kb disk_use_percent <<< $(df -k "$DISK_PATH" 2>/dev/null | awk 'NR==2 {gsub(/%/,"",$5); print $2, $3, $4, $5}')
    disk_total_gb=$(awk "BEGIN {printf \"%.2f\", $disk_total_kb / 1048576}")
    disk_used_gb=$(awk "BEGIN {printf \"%.2f\", $disk_used_kb / 1048576}")
    disk_avail_gb=$(awk "BEGIN {printf \"%.2f\", $disk_avail_kb / 1048576}")
    disk_percent=${disk_use_percent:-0}
    
    # Network I/O from /proc/net/dev (sum all interfaces except lo)
    net_rx=0
    net_tx=0
    while IFS=':' read -r iface stats; do
        iface=$(echo "$iface" | tr -d ' ')
        if [ "$iface" != "lo" ] && [ -n "$iface" ]; then
            read -r rx_bytes _ _ _ _ _ _ _ tx_bytes _ <<< "$stats"
            if [[ $rx_bytes =~ ^[0-9]+$ ]]; then
                net_rx=$((net_rx + rx_bytes))
                net_tx=$((net_tx + tx_bytes))
            fi
        fi
    done < /proc/net/dev
    
    # Convert bytes to MB
    if [ "$first_run" = true ]; then
        net_rx_mb=0
        net_tx_mb=0
    else
        net_rx_delta=$((net_rx - prev_net_rx))
        net_tx_delta=$((net_tx - prev_net_tx))
        net_rx_mb=$(awk "BEGIN {printf \"%.2f\", $net_rx_delta / 1048576}")
        net_tx_mb=$(awk "BEGIN {printf \"%.2f\", $net_tx_delta / 1048576}")
    fi
    prev_net_rx=$net_rx
    prev_net_tx=$net_tx
    
    first_run=false
    
    # Write metrics to stdout
    echo -e "${timestamp}\t${cpu_percent}\t${mem_used_mb}\t${mem_total_mb}\t${mem_percent}\t${disk_total_gb}\t${disk_used_gb}\t${disk_avail_gb}\t${disk_percent}\t${disk_read_mb}\t${disk_write_mb}\t${net_rx_mb}\t${net_tx_mb}"
    
    sleep "$INTERVAL"
done
