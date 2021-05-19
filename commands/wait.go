package commands

import (
	"fmt"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/martinlindhe/notify"
)

func Wait(h, iap, operation string, sleep int, alarm bool) error {
	cromwellClient := cromwell.New(h, iap)
	resp, err := cromwellClient.Status(operation)
	if err != nil {
		return err
	}
	status := resp.Status
	fmt.Printf("Status=%s\n", resp.Status)

	fmt.Printf("Time between status check = %d\n", sleep)
	for status == "Running" || status == "Submitted" {
		time.Sleep(time.Duration(sleep) * time.Second)
		resp, err := cromwellClient.Status(operation)
		if err != nil {
			return err
		}
		fmt.Printf("Status=%s\n", resp.Status)
		status = resp.Status
	}

	if alarm {
		notify.Alert("🐖 Cromwell Cli", "alert", "Your workflow ended", "")
	}
	return nil
}
