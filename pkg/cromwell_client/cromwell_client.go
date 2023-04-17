package cromwell_client

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

	"github.com/google/go-querystring/query"
)

type Client struct {
	Host   string
	Iap    string
	Logger *log.Logger
}

func New(h, t string) *Client {
	return &Client{
		Host:   h,
		Iap:    t,
		Logger: log.New(os.Stderr, "", log.LstdFlags),
	}
}

func (c *Client) Kill(o string) (SubmitResponse, error) {
	var sr SubmitResponse

	route := fmt.Sprintf("/api/workflows/v1/%s/abort", o)
	err := c.iapAwareRequest("POST", route, nil, nil, &sr)
	return sr, err
}

func (c *Client) Status(o string) (SubmitResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/status", o)
	var or SubmitResponse
	err := c.iapAwareRequest("GET", route, nil, nil, &or)
	return or, err

}

func (c *Client) Outputs(o string) (OutputsResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/outputs", o)
	var or OutputsResponse
	err := c.iapAwareRequest("GET", route, nil, nil, &or)
	return or, err
}

func (c *Client) Query(p *ParamsQueryGet) (QueryResponse, error) {
	route := "/api/workflows/v1/query"
	var qr QueryResponse
	err := c.iapAwareRequest("GET", route, p, nil, &qr)
	return qr, err

}

// Metadata uses the Cromwell Server metadata endpoint to get the metadata for a workflow
// Be aware of this limitation: https://github.com/broadinstitute/cromwell/issues/4124
func (c *Client) Metadata(o string, p *ParamsMetadataGet) (MetadataResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/metadata", o)
	var mr MetadataResponse
	err := c.iapAwareRequest("GET", route, p, nil, &mr)
	return mr, err
}

func (c *Client) Submit(requestFields *SubmitRequest) (SubmitResponse, error) {
	route := "/api/workflows/v1"
	fileParams := submitPrepare(*requestFields)
	var sr SubmitResponse
	err := c.iapAwareRequest("POST", route, nil, fileParams, &sr)
	return sr, err
}

func (c *Client) iapAwareRequest(method, route string, urlParams interface{}, files map[string]string, resp interface{}) error {
	var body bytes.Buffer
	var writer *multipart.Writer
	ct := "application/json"
	if files != nil {
		writer = c.prepareFormData(files, &body)
		ct = writer.FormDataContentType()
	}

	opts, err := query.Values(urlParams)
	if err != nil {
		return err
	}

	var uri string
	if len(opts) == 0 {
		uri = fmt.Sprintf("%s%s", c.Host, route)
	} else {
		uri = fmt.Sprintf("%s%s?%s", c.Host, route, opts.Encode())
	}
	req, err := http.NewRequest(method, uri, &body)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Content-Type", ct)
	r, err := c.makeRequest(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	err = json.NewDecoder(r.Body).Decode(resp)
	return err
}

func (c *Client) prepareFormData(files map[string]string, body *bytes.Buffer) *multipart.Writer {
	var (
		w = multipart.NewWriter(body)
	)

	for field, path := range files {

		filename := filepath.Base(path)

		fw, err := w.CreateFormFile(field, filename)
		if err != nil {
			log.Fatal(err)
		}

		file, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}

		if _, err := io.Copy(fw, file); err != nil {
			log.Fatal(err)
		}
	}

	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
	return w
}

func (c *Client) makeRequest(req *http.Request) (*http.Response, error) {
	if c.Iap != "" {
		token := getGoogleIapToken(c.Iap)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	log.Printf("%s request to: %s\n", req.Method, req.URL)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
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
	var er = ErrorResponse{
		HTTPStatus: r.Status,
	}
	if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
		log.Println("No json body in response")
	}
	return fmt.Errorf("submission failed. the server returned %#v", er)
}
