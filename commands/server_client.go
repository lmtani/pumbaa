package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
)

func New(h, t string) Client {
	return Client{host: h, token: t}
}

func FromInterface(i interface{}) Client {
	c := Client{}
	mapstructure.Decode(i, &c)
	return c
}

type Client struct {
	host  string
	token string
}

type ErrorResponse struct {
	HTTPStatus string
	Status     string
	Message    string
}

func (c *Client) makeRequest(req *http.Request) (*http.Response, error) {
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}
	zap.S().Debugw(fmt.Sprintf("%s request to: %s", req.Method, req.URL))
	client := &http.Client{}
	return client.Do(req)
}

func (c *Client) get(u string) (*http.Response, error) {
	uri := fmt.Sprintf("%s%s", c.host, u)
	req, _ := http.NewRequest("GET", uri, nil)
	return c.makeRequest(req)
}

func (c *Client) post(u string, files map[string]string) (*http.Response, error) {
	var (
		uri    = fmt.Sprintf("%s%s", c.host, u)
		body   = new(bytes.Buffer)
		writer = multipart.NewWriter(body)
	)

	for field, path := range files {
		// gets file name from file path
		filename := filepath.Base(path)
		// creates a new form file writer
		fw, err := writer.CreateFormFile(field, filename)
		if err != nil {
			return nil, err
		}

		// prepare the file to be read
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		// copies the file content to the form file writer
		if _, err := io.Copy(fw, file); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return c.makeRequest(req)
}

func (c *Client) Kill(o string) (SubmitResponse, error) {
	var sr SubmitResponse

	route := fmt.Sprintf("/api/workflows/v1/%s/abort", o)
	r, err := c.post(route, map[string]string{})
	if err != nil {
		return sr, err
	}
	defer r.Body.Close()

	if r.StatusCode >= 400 {
		var er = ErrorResponse{
			HTTPStatus: r.Status,
		}

		if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
			return sr, err
		}

		return sr, fmt.Errorf("Submission failed. The server returned %#v", er)
	}

	if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
		return sr, err
	}
	return sr, nil
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
	route := "/api/workflows/v1/query"
	var urlParams string
	if n != "" {
		urlParams = fmt.Sprintf("?name=%s", n)
	}
	var qr QueryResponse
	r, err := c.get(route + urlParams)
	if err != nil {
		return qr, err
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&qr); err != nil {
		return qr, err
	}

	if r.StatusCode >= 400 {
		return qr, fmt.Errorf("Submission failed. The server returned %d\n%#v", r.StatusCode, qr)
	}
	return qr, nil
}

func (c *Client) Metadata(o string) (MetadataResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/metadata", o)
	var mr MetadataResponse
	r, err := c.get(route)
	if err != nil {
		return mr, nil
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
		return mr, err
	}

	if r.StatusCode >= 400 {
		return mr, fmt.Errorf("Submission failed. The server returned %d\n%#v", r.StatusCode, mr)
	}
	return mr, nil
}

func submitPrepare(r SubmitRequest) map[string]string {
	fileParams := map[string]string{
		"workflowSource": r.workflowSource,
		"workflowInputs": r.workflowInputs,
	}
	if r.workflowDependencies != "" {
		fileParams["workflowDependencies"] = r.workflowDependencies
	}
	return fileParams
}

func (c *Client) Submit(requestFields SubmitRequest) (SubmitResponse, error) {
	route := "/api/workflows/v1"
	fileParams := submitPrepare(requestFields)
	var sr SubmitResponse
	r, err := c.post(route, fileParams)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
		return sr, err
	}

	if r.StatusCode >= 400 {
		return sr, fmt.Errorf("Submission failed. The server returned %d\n%#v", r.StatusCode, sr)
	}

	return sr, nil
}
