package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"go.uber.org/zap"
)

func New(h string) Client {
	return Client{host: h}
}

type Client struct {
	host string
}

func (c *Client) get(u string) (*http.Response, error) {
	uri := fmt.Sprintf("%s%s", c.host, u)
	zap.S().Debugw(fmt.Sprintf("Request to: %s", uri))
	r, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	// content, _ := ioutil.ReadAll(r.Body)
	// print(string(content))
	return r, nil
}

func (c *Client) post(u string, files map[string]string) (*http.Response, error) {
	uri := fmt.Sprintf("%s%s", c.host, u)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for k, v := range files {
		file, _ := os.Open(v)
		content, _ := ioutil.ReadAll(file)
		fi, _ := file.Stat()
		part, _ := writer.CreateFormFile(k, fi.Name())
		_, err := part.Write(content)
		if err != nil {
			return nil, err
		}
	}

	err := writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return r, nil
}

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

func (c *Client) Query(n string) (QueryResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/query")
	r, err := c.get(route)
	if err != nil {
		return QueryResponse{}, err
	}
	defer r.Body.Close()
	resp := QueryResponse{}

	body, _ := ioutil.ReadAll(r.Body)
	fmt.Println(string(body))
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return QueryResponse{}, err
	}
	fmt.Println(resp)
	return resp, nil
}

func (c *Client) Metadata(o string) string {
	route := fmt.Sprintf("/api/workflow/v1/%s/status", o)
	return route
}

func (c *Client) Submit(w, i, d string) error {
	route := "/api/workflows/v1"
	fileParams := map[string]string{
		"workflowSource":       w,
		"workflowInputs":       i,
		"workflowDependencies": d,
	}
	r, err := c.post(route, fileParams)
	if err != nil {
		log.Fatal(err)
	}
	// defer r.Body.Close()

	fmt.Println(r.Status)
	return nil
}

type ErrorResponse struct {
	status  string `json: "status"`
	message string `json: "message"`
}

type QueryResponse struct {
	Results           []QueryResponseWorkflow
	TotalResultsCount int
}

type QueryResponseWorkflow struct {
	ID                    string
	Name                  string
	Status                string
	Submission            string
	Start                 string
	End                   string
	MetadataArchiveStatus string
}
