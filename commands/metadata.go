package commands

import "fmt"

func MetadataWorkflow(c Client, o string) error {
	resp, err := c.Metadata(o)
	if err != nil {
		return err
	}
	fmt.Println(resp)
	return err
}
