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

// queryCmd represents the query command
var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		WorkflowProvider := cromwell.NewCromwellWorkflowProvider("http://localhost:8000")
		usecase := &usecases.QueryWorkflows{
			WorkflowProvider: WorkflowProvider,
		}
		workflows, err := usecase.Execute()
		if err != nil {
			fmt.Println("Error querying workflows:", err)
			return
		}

		workflowFormatter := table.NewWorkflowTableFormatter()
		err = workflowFormatter.Query(workflows.Workflows)
		if err != nil {
			fmt.Println("Error formatting workflows:", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
	// queryCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
