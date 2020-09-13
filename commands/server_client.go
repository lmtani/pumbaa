package commands

import (
	"bytes"
	"encoding/json"
	"errors"
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

func (c *Client) Kill(o string) (SubmitResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/abort", o)
	r, err := c.post(route, map[string]string{})
	if err != nil {
		return SubmitResponse{}, err
	}
	defer r.Body.Close()
	body, _ := ioutil.ReadAll(r.Body)
	if r.StatusCode >= 400 {
		msg := fmt.Sprintf("Submission failed. The server returned %d\n%s", r.StatusCode, body)
		return SubmitResponse{}, errors.New(msg)
	}
	resp := SubmitResponse{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return SubmitResponse{}, err
	}
	return resp, nil
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
	r, err := c.get(route + urlParams)
	if err != nil {
		return QueryResponse{}, err
	}
	defer r.Body.Close()
	resp := QueryResponse{}

	body, _ := ioutil.ReadAll(r.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return QueryResponse{}, err
	}
	return resp, nil
}

func (c *Client) Metadata(o string) string {
	route := fmt.Sprintf("/api/workflow/v1/%s/status", o)
	return route
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

	r, err := c.post(route, fileParams)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Body.Close()
	resp := SubmitResponse{}
	body, _ := ioutil.ReadAll(r.Body)
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return SubmitResponse{}, err
	}
	if r.StatusCode >= 400 {
		msg := fmt.Sprintf("Submission failed. The server returned %d\n%s", r.StatusCode, body)
		return SubmitResponse{}, errors.New(msg)
	}

	return resp, nil
}
