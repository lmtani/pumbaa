package cromwell

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

func New(h, t string) Client {
	return Client{
		Host:   h,
		Iap:    t,
		Logger: log.New(os.Stderr, "", log.LstdFlags),
	}
}

func Default() Client {
	return Client{
		Host:   "http://127.0.0.1:8000",
		Iap:    "",
		Logger: log.New(os.Stderr, "", log.LstdFlags),
	}
}

func (c *Client) Kill(o string) (SubmitResponse, error) {
	var sr SubmitResponse

	route := fmt.Sprintf("/api/workflows/v1/%s/abort", o)
	err := c.iapAwareRequest("GET", route, nil, nil, &sr)
	if err != nil {
		return sr, err
	}
	return sr, nil
}

func (c *Client) Status(o string) (SubmitResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/status", o)
	var or SubmitResponse
	err := c.iapAwareRequest("GET", route, nil, nil, &or)
	if err != nil {
		return or, err
	}
	return or, nil

}

func (c *Client) Outputs(o string) (OutputsResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/outputs", o)
	var or OutputsResponse
	err := c.iapAwareRequest("GET", route, nil, nil, &or)
	if err != nil {
		return or, err
	}
	return or, nil
}

func (c *Client) Query(p ParamsQueryGet) (QueryResponse, error) {
	route := "/api/workflows/v1/query"
	var qr QueryResponse
	err := c.iapAwareRequest("GET", route, p, nil, &qr)
	if err != nil {
		return qr, err
	}
	return qr, nil

}

// Metadata uses the Cromwell Server metadata endpoint to get the metadata for a workflow
// Be aware of this limitation: https://github.com/broadinstitute/cromwell/issues/4124
func (c *Client) Metadata(o string, p ParamsMetadataGet) (MetadataResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/metadata", o)
	var mr MetadataResponse
	err := c.iapAwareRequest("GET", route, p, nil, &mr)
	if err != nil {
		return mr, err
	}
	return mr, nil
}

func (c *Client) Submit(requestFields SubmitRequest) (SubmitResponse, error) {
	route := "/api/workflows/v1"
	fileParams := submitPrepare(requestFields)
	var sr SubmitResponse
	err := c.iapAwareRequest("GET", route, nil, fileParams, &sr)
	if err != nil {
		return sr, err
	}
	return sr, nil
}

func (c *Client) iapAwareRequest(method, route string, urlParams interface{}, files map[string]string, resp interface{}) error {
	var body bytes.Buffer
	var writer *multipart.Writer
	ct := "application/json"
	if files != nil {
		writer = c.prepareFormData(files, body)
		ct = writer.FormDataContentType()
	}

	opts, err := query.Values(urlParams)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("%s%s?%s", c.Host, route, opts.Encode())
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
	if err := json.NewDecoder(r.Body).Decode(resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) prepareFormData(files map[string]string, body bytes.Buffer) *multipart.Writer {
	var (
		w = multipart.NewWriter(&body)
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
