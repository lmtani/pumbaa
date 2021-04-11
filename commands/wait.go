package commands

import (
	"fmt"
	"time"

	"github.com/lmtani/cromwell-cli/pkg/cromwell"
	"github.com/martinlindhe/notify"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func Wait(c *cli.Context) error {
	cromwellClient := cromwell.New(c.String("host"), c.String("iap"))
	resp, err := cromwellClient.Status(c.String("operation"))
	if err != nil {
		return err
	}
	status := resp.Status
	zap.S().Info(fmt.Sprintf("Status=%s", resp.Status))

	seconds := c.Int("sleep")
	zap.S().Info(fmt.Sprintf("Time between status check = %d", seconds))
	for status == "Running" {
		time.Sleep(time.Duration(seconds) * time.Second)
		resp, err := cromwellClient.Status(c.String("operation"))
		if err != nil {
			return err
		}
		zap.S().Info(fmt.Sprintf("Status=%s", resp.Status))
		status = resp.Status
	}

	if c.Bool("alarm") {
		notify.Alert("🐖 Cromwell Cli", "alert", "Your workflow ended", "")
	}
	return nil
}
