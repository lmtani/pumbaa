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
)

type Client struct {
	host string
	iap  string
}

func New(h, t string) Client {
	return Client{host: h, iap: t}
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

func (c *Client) Query(p url.Values) (QueryResponse, error) {
	route := "/api/workflows/v1/query"
	var qr QueryResponse
	r, err := c.get(route + "?" + p.Encode())
	if err != nil {
		return qr, err
	}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&qr); err != nil {
		return qr, err
	}

	return qr, nil
}

func (c *Client) Metadata(o string, p url.Values) (MetadataResponse, error) {
	route := fmt.Sprintf("/api/workflows/v1/%s/metadata"+"?"+p.Encode(), o)
	var mr MetadataResponse
	r, err := c.get(route)
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
	return fmt.Errorf("Submission failed. The server returned %#v", er)
}
