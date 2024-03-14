package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-querystring/query"
	"github.com/lmtani/pumbaa/internal/types"
	"google.golang.org/api/idtoken"
)

type Client struct {
	Host   string
	Iap    string
	Logger *log.Logger
}

func NewCromwellClient(h, t string) *Client {
	return &Client{
		Host:   h,
		Iap:    t,
		Logger: log.New(os.Stderr, "", log.LstdFlags),
	}
}

func (c *Client) Kill(o string) (types.SubmitResponse, error) {
	var sr types.SubmitResponse

	route := fmt.Sprintf("/api/workflows/v1/%s/abort", o)
	err := c.iapAwareRequest("POST", route, nil, nil, &sr)
	return sr, err
}

func (c *Client) Status(o string) (types.SubmitResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/status", o)
	var or types.SubmitResponse
	err := c.iapAwareRequest("GET", route, nil, nil, &or)
	return or, err

}

func (c *Client) Outputs(o string) (types.OutputsResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/outputs", o)
	var or types.OutputsResponse
	err := c.iapAwareRequest("GET", route, nil, nil, &or)
	return or, err
}

func (c *Client) Query(p *types.ParamsQueryGet) (types.QueryResponse, error) {
	route := "/api/workflows/v1/query"
	var qr types.QueryResponse
	err := c.iapAwareRequest("GET", route, p, nil, &qr)
	return qr, err

}

// Metadata uses the Cromwell Server metadata endpoint to get the metadata for a workflow
// Be aware of this limitation: https://github.com/broadinstitute/cromwell/issues/4124
func (c *Client) Metadata(o string, p *types.ParamsMetadataGet) (types.MetadataResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/metadata", o)
	var mr types.MetadataResponse
	err := c.iapAwareRequest("GET", route, p, nil, &mr)
	return mr, err
}

func (c *Client) Submit(requestFields *types.SubmitRequest) (types.SubmitResponse, error) {
	route := "/api/workflows/v1"
	fileParams := submitPrepare(*requestFields)
	var sr types.SubmitResponse
	err := c.iapAwareRequest("POST", route, nil, fileParams, &sr)
	return sr, err
}

func (c *Client) iapAwareRequest(method, route string, urlParams interface{}, files map[string]string, resp interface{}) error {
	var body bytes.Buffer

	ct := "application/json"
	if files != nil {
		writer, err := c.prepareFormData(files, &body)
		if err != nil {
			return err
		}
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
	var w = multipart.NewWriter(body)

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
	if c.Iap != "" {
		token := getGoogleIapToken(c.Iap)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
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
	var er = types.ErrorResponse{
		HTTPStatus: r.Status,
	}
	if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
		log.Println("No json body in response")
	}
	return fmt.Errorf("submission failed. the server returned %#v", er)
}

// Can be decoupled
func submitPrepare(r types.SubmitRequest) map[string]string {
	fileParams := map[string]string{
		"workflowSource": r.WorkflowSource,
		"workflowInputs": r.WorkflowInputs,
	}
	if r.WorkflowDependencies != "" {
		fileParams["workflowDependencies"] = r.WorkflowDependencies
	}
	if r.WorkflowOptions != "" {
		fileParams["workflowOptions"] = r.WorkflowOptions
	}
	return fileParams
}

// Can be decoupled
func getGoogleIapToken(aud string) string {
	ctx := context.Background()
	ts, err := idtoken.NewTokenSource(ctx, aud)
	if err != nil {
		log.Fatal(err)
	}
	token, err := ts.Token()
	if err != nil {
		log.Fatal(err)
	}
	return token.AccessToken
}

type FakeCromwell struct {
	MetadataCalled bool
	AbortCalled    bool
	StatusCalled   bool
	OutputsCalled  bool
	QueryCalled    bool
	SubmitCalled   bool

	SubmitResponse   types.SubmitResponse
	MetadataResponse types.MetadataResponse
	OutputsResponse  types.OutputsResponse
	QueryResponse    types.QueryResponse

	Err error
}

func (f *FakeCromwell) Metadata(o string, p *types.ParamsMetadataGet) (types.MetadataResponse, error) {
	f.MetadataCalled = true
	return f.MetadataResponse, f.Err
}

func (f *FakeCromwell) Kill(o string) (types.SubmitResponse, error) {
	f.AbortCalled = true
	return f.SubmitResponse, f.Err
}

func (f *FakeCromwell) Status(o string) (types.SubmitResponse, error) {
	f.StatusCalled = true
	return f.SubmitResponse, f.Err
}

func (f *FakeCromwell) Outputs(o string) (types.OutputsResponse, error) {
	f.OutputsCalled = true
	return f.OutputsResponse, f.Err
}

func (f *FakeCromwell) Query(p *types.ParamsQueryGet) (types.QueryResponse, error) {
	f.QueryCalled = true
	return f.QueryResponse, f.Err
}

func (f *FakeCromwell) Submit(requestFields *types.SubmitRequest) (types.SubmitResponse, error) {
	f.SubmitCalled = true
	return f.SubmitResponse, f.Err
}