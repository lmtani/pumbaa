package commands

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

func New(h string) Client {
	return Client{host: h}
}

type Client struct {
	host string
}

func (c *Client) get(u string, target interface{}) error {
	uri := fmt.Sprintf("%s%s", c.host, u)
	zap.S().Debugw(fmt.Sprintf("Request to: %s", uri))
	r, err := http.Get(uri)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

// func (c *Client) post(u string, target interface{}) error {
// 	uri := fmt.Sprintf("%s%s", c.host, u)
// 	zap.S().Debugw(fmt.Sprintf("Request to: %s", uri))
// 	return nil
// }

func (c *Client) Kill(o string) string {
	route := fmt.Sprintf("/api/workflow/v1/%s/abort", o)
	return route
}

func (c *Client) Status(o string) string {
	route := fmt.Sprintf("/api/workflow/v1/%s/status", o)
	return route
}

func (c *Client) Outputs(o string) string {
	route := fmt.Sprintf("/api/workflow/v1/%s/status", o)
	return route
}

func (c *Client) Query(n string) (*QueryResponse, error) {
	t := new(QueryResponse)
	route := fmt.Sprintf("/api/workflows/v1/query?name=%s", n)
	err := c.get(route, t)
	if err != nil {
		return t, err
	}
	return t, nil
}

func (c *Client) Metadata(o string) string {
	route := fmt.Sprintf("/api/workflow/v1/%s/status", o)
	return route
}

func (c *Client) Submit() string {
	route := "/api/workflows/v1"
	return route
}
