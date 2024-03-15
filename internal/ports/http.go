package ports

import "net/http"

type HTTPClient interface {
	Get(url string) (resp *http.Response, err error)
	Head(url string) (resp *http.Response, err error)
	DownloadWithProgress(url string, fileName string) error
}
