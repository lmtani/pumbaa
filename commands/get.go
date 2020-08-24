package commands

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"go.uber.org/zap"
)

func GetWorkflow() error {
	resp, err := http.Get("https://httpbin.org/get")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	zap.S().Infow("An info message", "Key1", "value1", "Key2", 1)
	return nil
}
