package commands

import (
	"math/rand"
	"net/http"
	"net/http/httptest"

	"log"
)

func buildTestServerMutable(url string) *httptest.Server {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == url {
				resp := `{"id": "a-new-uuid", "status": "Done"}`
				_, err := w.Write([]byte(resp))
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
	// Status=Done
	// Time between status check = 1
}
