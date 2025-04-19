package cromwell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/google/go-querystring/query"
)

type Client struct {
	Host   string
	Logger *log.Logger
}

func NewCromwellClient(h string) *Client {
	return &Client{
		Host:   h,
		Logger: log.New(os.Stderr, "", log.LstdFlags),
	}
}

func (c *Client) Kill(o string) (SubmitResponse, error) {
	var sr SubmitResponse

	route := fmt.Sprintf("/api/workflows/v1/%s/abort", o)
	err := c.request("POST", route, nil, nil, &sr)
	return sr, err
}

func (c *Client) Status(o string) (SubmitResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/status", o)
	var or SubmitResponse
	err := c.request("GET", route, nil, nil, &or)
	return or, err
}

func (c *Client) Outputs(o string) (OutputsResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/outputs", o)
	var or OutputsResponse
	err := c.request("GET", route, nil, nil, &or)
	return or, err
}

func (c *Client) Query(p interface{}) (QueryResponse, error) {
	route := "/api/workflows/v1/query"
	var qr QueryResponse
	err := c.request("GET", route, p, nil, &qr)
	return qr, err
}

// Metadata uses the Cromwell Server metadata endpoint to get the metadata for a workflow
// Be aware of this limitation: https://github.com/broadinstitute/cromwell/issues/4124
func (c *Client) Metadata(o string, urlParams interface{}) (MetadataResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/metadata", o)
	var mr MetadataResponse
	err := c.request("GET", route, urlParams, nil, &mr)
	return mr, err
}

func (c *Client) Submit(wdl, inputs, dependencies, options string) (SubmitResponse, error) {
	route := "/api/workflows/v1"
	fileParams := map[string]string{
		"workflowSource": wdl,
	}
	if inputs != "" {
		fileParams["workflowInputs"] = inputs
	}
	if dependencies != "" {
		fileParams["workflowDependencies"] = dependencies
	}
	if options != "" {
		fileParams["workflowOptions"] = options
	}
	var sr SubmitResponse
	err := c.request("POST", route, nil, fileParams, &sr)
	return sr, err
}

func (c *Client) request(method, route string, urlParams interface{}, files map[string]string, resp interface{}) error {
	var body bytes.Buffer

	ct := "application/json"
	if files != nil {
		writer, err := c.prepareFormData(files, &body)
		if err != nil {
			return err
		}
		ct = writer.FormDataContentType()
	}

	var opts url.Values
	if urlParams != nil {
		switch params := urlParams.(type) {
		case map[string]string:
			opts = url.Values{}
			for key, value := range params {
				opts.Add(key, value)
			}
		default:
			var err error
			opts, err = query.Values(urlParams)
			if err != nil {
				return err
			}
		}
	} else {
		opts = url.Values{}
	}

	var uri string
	if len(opts) == 0 {
		uri = fmt.Sprintf("%s%s", c.Host, route)
	} else {
		uri = fmt.Sprintf("%s%s?%s", c.Host, route, opts.Encode())
	}
	req, err := http.NewRequest(method, uri, &body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", ct)
	r, err := c.makeRequest(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			println(err)
		}
	}(r.Body)
	err = json.NewDecoder(r.Body).Decode(resp)
	return err
}

func (*Client) prepareFormData(files map[string]string, body *bytes.Buffer) (*multipart.Writer, error) {
	w := multipart.NewWriter(body)

	for field, path := range files {

		filename := filepath.Base(path)

		fw, err := w.CreateFormFile(field, filename)
		if err != nil {
			return w, err
		}

		file, err := os.Open(path)
		if err != nil {
			return w, err
		}

		if _, err := io.Copy(fw, file); err != nil {
			return w, err
		}
	}

	if err := w.Close(); err != nil {
		return w, err
	}
	return w, nil
}

func (c *Client) makeRequest(req *http.Request) (*http.Response, error) {
	log.Printf("%s request to: %s\n", req.Method, req.URL)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return &http.Response{}, err
	}
	if resp.StatusCode >= 400 {
		err := errorHandler(resp)
		if err != nil {
			return resp, err
		}
	}
	return resp, nil
}

func errorHandler(r *http.Response) error {
	er := ErrorResponse{
		HTTPStatus: r.Status,
	}
	if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
		log.Println("No json body in response")
	}
	return fmt.Errorf("submission failed. the server returned %#v", er)
}
