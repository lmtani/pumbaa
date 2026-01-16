# Resource Analysis

Pumbaa provides a suite for analyzing workflow resource efficiency. It helps you understand how your tasks consume CPU, Memory, and Disk, and identifies optimization opportunities.

## Individual Workflow Analysis

The `resource-report` command analyzes the monitoring logs for a specific workflow execution.

### Prerequisites

To use this feature, your workflow instructions must adhere to the following requirements:

1.  **Monitoring Script**: The workflow must have been executed with Cromwell options that enable the resource monitoring script. See [Resource Monitoring](resource-monitoring.md) for setup instructions.
2.  **Input Availability**: The input files for the tasks must still be accessible (e.g., on GCS or local disk) so Pumbaa can calculate their sizes to correlate with resource usage.

### Usage

```bash
pumbaa workflow resource-report [flags] <workflow-id>
```

**Flags:**

-   `--concurrency/-c`: Number of concurrent workers for fetching monitoring logs (default: 5).

### Features

-   **Utilization Engine**: Parses monitoring TSVs to compute Peak, Mean, and Efficiency metrics.
-   **Recursive Input Analysis**: Calculates the total footprint of input data to help correlate data size with resource usage.
-   **Optimization Recommendations**: Automatically flags inefficient tasks (e.g., low CPU utilization, high memory waste) and suggests tuning actions.

## Consolidated Visualization

The `analyze resources` command generates an interactive HTML dashboard from a collection of resource report TSVs.

### Usage

```bash
pumbaa analyze resources [flags] <directory>
```

**Flags:**

-   `--output/-o`: Output HTML file path (default: `resource_report.html`).

### Features

-   **Interactive Dashboard**: A self-contained HTML report with interactive charts.
-   **Efficiency Scatter Plots**: Visualizes Efficiency vs. Input Size for CPU, Memory, and Disk to spot scaling trends and outliers.
-   **Smart Filtering**: Filter tasks by status (Critical, Warning, Good) to focus on the biggest cost drivers.
