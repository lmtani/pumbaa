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
	Host string
	Iap  string
}

func New(h, t string) Client {
	return Client{Host: h, Iap: t}
}

func Default() Client {
	return Client{Host: "http://127.0.0.1:8000", Iap: ""}
}

func (c *Client) Setup(h, t string) {
	c.Host = h
	c.Iap = t
}

func (c *Client) Kill(o string) (SubmitResponse, error) {
	var sr SubmitResponse

	route := fmt.Sprintf("/api/workflows/v1/%s/abort", o)
	r, err := c.post(route, map[string]string{})
	if err != nil {
		return sr, err
	}
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
		return sr, err
	}
	return sr, nil
}

func (c *Client) Status(o string) (SubmitResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/status", o)
	var sr SubmitResponse
	r, err := c.get(route)
	if err != nil {
		return sr, err
	}
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
		return sr, err
	}
	return sr, nil
}

func (c *Client) Outputs(o string) (OutputsResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/outputs", o)
	var or = OutputsResponse{}
	r, err := c.get(route)
	if err != nil {
		return or, err
	}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&or); err != nil {
		return or, err
	}

	return or, nil
}

func (c *Client) Query(p ParamsQueryGet) (QueryResponse, error) {
	route := "/api/workflows/v1/query"

	opts, err := query.Values(p)
	if err != nil {
		return QueryResponse{}, err
	}

	var qr QueryResponse
	r, err := c.get(route + "?" + opts.Encode())
	if err != nil {
		return qr, err
	}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&qr); err != nil {
		return qr, err
	}

	return qr, nil
}

// Metadata uses the Cromwell Server metadata endpoint to get the metadata for a workflow
// Be aware of this limitation: https://github.com/broadinstitute/cromwell/issues/4124
func (c *Client) Metadata(o string, p ParamsMetadataGet) (MetadataResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/metadata", o)

	opts, err := query.Values(p)
	if err != nil {
		return MetadataResponse{}, err
	}
	var mr MetadataResponse
	r, err := c.get(route + "?" + opts.Encode())
	if err != nil {
		return mr, err
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
		return mr, err
	}

	return mr, nil
}

func (c *Client) Submit(requestFields SubmitRequest) (SubmitResponse, error) {
	route := "/api/workflows/v1"
	fileParams := submitPrepare(requestFields)
	var sr SubmitResponse
	r, err := c.post(route, fileParams)
	if err != nil {
		return sr, err
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
		return sr, err
	}

	return sr, nil
}

func (c *Client) get(u string) (*http.Response, error) {
	uri := fmt.Sprintf("%s%s", c.Host, u)
	req, _ := http.NewRequest("GET", uri, nil)
	return c.makeRequest(req)
}

func (c *Client) post(u string, files map[string]string) (*http.Response, error) {
	var (
		uri    = fmt.Sprintf("%s%s", c.Host, u)
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
