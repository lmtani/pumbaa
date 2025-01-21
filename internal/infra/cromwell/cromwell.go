package cromwell

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

	"github.com/lmtani/pumbaa/internal/entities"

	"github.com/google/go-querystring/query"
)

type Cromwell struct {
	Host   string
	Gcp    entities.GoogleCloudPlatform
	Logger *log.Logger
	aud    string
}

func NewCromwellClient(h string, gcp entities.GoogleCloudPlatform) *Cromwell {
	return &Cromwell{
		Host:   h,
		Gcp:    gcp,
		Logger: log.New(os.Stderr, "", log.LstdFlags),
	}
}

func (c *Cromwell) Kill(o string) (entities.SubmitResponse, error) {
	var sr entities.SubmitResponse

	route := fmt.Sprintf("/api/workflows/v1/%s/abort", o)
	err := c.iapAwareRequest("POST", route, nil, nil, &sr)
	return sr, err
}

func (c *Cromwell) Status(o string) (entities.SubmitResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/status", o)
	var or entities.SubmitResponse
	err := c.iapAwareRequest("GET", route, nil, nil, &or)
	return or, err
}

func (c *Cromwell) Outputs(o string) (entities.OutputsResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/outputs", o)
	var or entities.OutputsResponse
	err := c.iapAwareRequest("GET", route, nil, nil, &or)
	return or, err
}

func (c *Cromwell) Query(p *entities.ParamsQueryGet) (entities.QueryResponse, error) {
	route := "/api/workflows/v1/query"
	var qr entities.QueryResponse
	err := c.iapAwareRequest("GET", route, p, nil, &qr)
	return qr, err
}

// Metadata uses the Cromwell Server metadata endpoint to get the metadata for a workflow
// Be aware of this limitation: https://github.com/broadinstitute/cromwell/issues/4124
func (c *Cromwell) Metadata(o string, p *entities.ParamsMetadataGet) (entities.MetadataResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/metadata", o)
	var mr entities.MetadataResponse
	err := c.iapAwareRequest("GET", route, p, nil, &mr)
	return mr, err
}

func (c *Cromwell) Submit(wdl, inputs, dependencies, options string) (entities.SubmitResponse, error) {
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
	var sr entities.SubmitResponse
	err := c.iapAwareRequest("POST", route, nil, fileParams, &sr)
	return sr, err
}

func (c *Cromwell) iapAwareRequest(method, route string, urlParams interface{}, files map[string]string, resp interface{}) error {
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

func (*Cromwell) prepareFormData(files map[string]string, body *bytes.Buffer) (*multipart.Writer, error) {
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

func (c *Cromwell) setAudience(aud string) {
	c.aud = aud
}

func (c *Cromwell) makeRequest(req *http.Request) (*http.Response, error) {
	if c.Gcp != nil {
		ctx := context.Background()
		token, err := c.Gcp.GetIAPToken(ctx, c.aud)
		if err != nil {
			return &http.Response{}, err
		}
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
	er := entities.ErrorResponse{
		HTTPStatus: r.Status,
	}
	if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
		log.Println("No json body in response")
	}
	return fmt.Errorf("submission failed. the server returned %#v", er)
}
