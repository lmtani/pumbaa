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
	"net/url"
	"os"
	"path/filepath"

	"google.golang.org/api/idtoken"
)

type Client struct {
	host string
	iap  string
}

func New(h, t string) Client {
	return Client{host: h, iap: t}
}

func (c *Client) get(u string) *http.Response {
	uri := fmt.Sprintf("%s%s", c.host, u)
	req, _ := http.NewRequest("GET", uri, nil)
	return c.makeRequest(req)
}

func (c *Client) post(u string, files map[string]string) *http.Response {
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
			log.Fatal(err)
		}

		// prepare the file to be read
		file, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}

		// copies the file content to the form file writer
		if _, err := io.Copy(fw, file); err != nil {
			log.Fatal(err)
		}
	}

	if err := writer.Close(); err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return c.makeRequest(req)
}

func (c *Client) makeRequest(req *http.Request) *http.Response {
	if c.iap != "" {
		token := getGoogleIapToken(c.iap)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	log.Printf("%s request to: %s\n", req.Method, req.URL)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode >= 400 {
		errorHandler(resp)
	}
	return resp
}

func (c *Client) Kill(o string) (SubmitResponse, error) {
	var sr SubmitResponse

	route := fmt.Sprintf("/api/workflows/v1/%s/abort", o)
	r := c.post(route, map[string]string{})
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
		return sr, err
	}
	return sr, nil
}

func (c *Client) Status(o string) (SubmitResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/status", o)
	var sr SubmitResponse
	r := c.get(route)
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
		return sr, err
	}
	return sr, nil
}

func (c *Client) Outputs(o string) (OutputsResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/outputs", o)
	r := c.get(route)
	var or = OutputsResponse{}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&or); err != nil {
		return or, err
	}

	return or, nil
}

func (c *Client) Query(p url.Values) (QueryResponse, error) {
	route := "/api/workflows/v1/query"
	var qr QueryResponse
	r := c.get(route + "?" + p.Encode())

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&qr); err != nil {
		return qr, err
	}

	if r.StatusCode >= 400 {
		return qr, fmt.Errorf("Submission failed. The server returned %d\n%#v", r.StatusCode, qr)
	}
	return qr, nil
}

func (c *Client) Metadata(o string, p url.Values) (MetadataResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/metadata"+"?"+p.Encode(), o)
	var mr MetadataResponse
	r := c.get(route)

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
		return mr, err
	}

	if r.StatusCode >= 400 {
		return mr, fmt.Errorf("Submission failed. The server returned: %d\n%#v", r.StatusCode, mr)
	}
	return mr, nil
}

func (c *Client) Submit(requestFields SubmitRequest) (SubmitResponse, error) {
	route := "/api/workflows/v1"
	fileParams := submitPrepare(requestFields)
	var sr SubmitResponse
	r := c.post(route, fileParams)

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
		return sr, err
	}

	if r.StatusCode >= 400 {
		return sr, fmt.Errorf("Submission failed. The server returned %d\n%#v", r.StatusCode, sr)
	}

	return sr, nil
}

func submitPrepare(r SubmitRequest) map[string]string {
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

func errorHandler(r *http.Response) {
	var er = ErrorResponse{
		HTTPStatus: r.Status,
	}
	if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
		log.Fatal(err)
	}
	log.Fatalf("Submission failed. The server returned %#v", er)
}

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
