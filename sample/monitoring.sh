#!/bin/bash

MEM_TOTAL_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
MEM_TOTAL=$((MEM_TOTAL_KB/1024))
DISK_TOTAL=$(df -m | awk '$NF=="/cromwell_root"{printf "%d\n", $2/1024}')
echo "# TOTAL_DISK_GB:${DISK_TOTAL}"
echo "# TOTAL_MEMORY_MB:${MEM_TOTAL}"
echo -e "TIMESTAMP\t%CPU\t%MEM\t%DISK"
while :
do
    CPU=$(awk '{u=$2+$4; t=$2+$4+$5; if (NR==1){u1=u; t1=t;} else print ($2+$4-u1) * 100 / (t-t1); }' <(grep 'cpu ' /proc/stat) <(sleep 1;grep 'cpu ' /proc/stat))
    DISK=$(df -m | awk '$NF=="/cromwell_root"{printf "%d\n", ($3/1024)/($2/1024) * 100}')

    MEM_AVAILABLE=$(grep MemAvailable /proc/meminfo | awk '{print $2}')
    MEM_AVAILABLE_MB=$((MEM_AVAILABLE/1024))
    MEM_USED=$((MEM_TOTAL - MEM_AVAILABLE_MB))
    MEM=$(echo "$MEM_USED $MEM_TOTAL" | awk '{print ($1/$2) * 100}')
    TIMESTAMP=$(date +"%T" )
	echo -e "$TIMESTAMP\t$CPU\t$MEM\t$DISK"
    sleep 5
done
