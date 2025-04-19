/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/lmtani/pumbaa/internal/infrastructure/workflow_formatter/table"
	"github.com/lmtani/pumbaa/internal/infrastructure/workflow_provider/cromwell"
	"github.com/lmtani/pumbaa/internal/usecases"
	"github.com/spf13/cobra"
)

// summaryCmd represents the summary command
var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Generates a simple report with some details about the pipeline",
	Long: `Generates a simple report with some details about the pipeline
	- Name
	- Status
	- Started At
	- Finished At`,
	Example: `pumbaa summary --o <uuid>`,
	Run: func(cmd *cobra.Command, args []string) {
		host, err := cmd.Flags().GetString("host")
		if err != nil {
			fmt.Println("Error fetching host:", err)
			return
		}
		workflowProvider := cromwell.NewCromwellWorkflowProvider(host)
		if workflowProvider == nil {
			fmt.Println("Error initializing workflow provider")
			return
		}
		summary := usecases.ReportWorkflow{
			WorkflowProvider: workflowProvider,
		}
		uuid, err := cmd.Flags().GetString("operation")
		if err != nil {
			fmt.Println("Error fetching UUID:", err)
			return
		}

		summaryResult, err := summary.Execute(uuid)
		if err != nil {
			fmt.Println("Error executing summary:", err)
			return
		}

		workflowFormatter := table.NewWorkflowTableFormatter()
		if err := workflowFormatter.Report(summaryResult); err != nil {
			fmt.Println("Error formatting summary:", err)
			return
		}
	},
}

func init() {
	summaryCmd.Flags().StringP("operation", "o", "", "UUID of the workflow to be summarized")
	err := summaryCmd.MarkFlagRequired("operation")
	if err != nil {
		fmt.Println("Error marking operation flag as required:", err)
		return
	}
	// Optional flag with host defaulting to "localhost"
	summaryCmd.Flags().StringP("host", "H", "http://localhost:8000", "Host of the Cromwell server")
	rootCmd.AddCommand(summaryCmd)
}
