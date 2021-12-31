package util

import (
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/lmtani/cromwell-cli/internal/prompt"
)

func BuildTestServer(url, resp string) *httptest.Server {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == url {
				_, err := w.Write([]byte(resp))
				if err != nil {
					log.Fatal(err)
				}
			}
		}))
	return ts
}

func BuildTestServerMutable(url string) *httptest.Server {
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

type promptForTests struct {
	byKeyReturn   string
	byIndexReturn int
}

func (p promptForTests) SelectByKey(taskOptions []string) (string, error) {
	return p.byKeyReturn, nil
}

func (p promptForTests) SelectByIndex(t prompt.TemplateOptions, sfn func(input string, index int) bool, items interface{}) (int, error) {
	return p.byIndexReturn, nil
}

func NewForTests(byKey string, byIndex int) *promptForTests {
	return &promptForTests{
		byKeyReturn: byKey, byIndexReturn: byIndex,
	}
}
