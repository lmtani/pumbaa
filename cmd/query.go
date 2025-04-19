package cmd

import (
	"fmt"

	"github.com/lmtani/pumbaa/internal/entities"
	workflowformatter "github.com/lmtani/pumbaa/internal/infrastructure/workflow_formatter"
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

		jsonFlag, err := cmd.Flags().GetBool("json")
		if err != nil {
			fmt.Println("Error getting json flag:", err)
			return
		}

		output := map[bool]entities.FormatType{
			true:  entities.JSONFormat,
			false: entities.TableFormat,
		}
		formatter := workflowformatter.GetFormatter(output[jsonFlag], nil)
		err = formatter.Query(workflows.Workflows)
		if err != nil {
			fmt.Println("Error formatting workflows:", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)
	queryCmd.Flags().Bool("json", false, "Output in JSON format")
	// queryCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
