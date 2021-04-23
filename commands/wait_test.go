package commands

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"log"
)

func buildTestServerMutable(url string) *httptest.Server {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == url {
				resps := []string{
					`{"id": "a-new-uuid", "status": "Running"}`,
					`{"id": "a-new-uuid", "status": "Done"}`,
				}
				_, err := w.Write([]byte(resps[rand.Intn(2)]))
				if err != nil {
					log.Fatal(err)
				}
			}
		}))
	return ts
}

func ExampleWait() {
	// Mock http server
	rand.Seed(3)
	operation := "aaaa-bbbb-uuid"
	ts := buildTestServerMutable("/api/workflows/v1/" + operation + "/status")
	defer ts.Close()

	err := Wait(ts.URL, "", operation, 1, false)
	if err != nil {
		log.Printf("Error: %#v", err)
	}

	// Output:
	// Status=Running
	// Time between status check = 1
	// Status=Done
}

func TestWait(t *testing.T) {
	// Mock http server
	operation := "aaaa-bbbb-uuid"
	ts := buildTestServer("/api/workflows/v1/"+operation+"/status", `{"id": "a-new-uuid", "status": "Submitted"}`)
	defer ts.Close()

	err := Wait(ts.URL, "", operation, 1, false)
	if err != nil {
		t.Error("Not found error expected, nil returned")
	}
}
