package adapters

import (
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/schollz/progressbar/v3"
)

type DefaultHTTP struct {
	http *http.Client
}

func NewDefaultHTTP() *DefaultHTTP {
	return &DefaultHTTP{
		http: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (d *DefaultHTTP) Get(url string) (resp *http.Response, err error) {
	return d.http.Get(url)
}

func (d *DefaultHTTP) Head(url string) (resp *http.Response, err error) {
	return d.http.Head(url)
}

func (d *DefaultHTTP) DownloadWithProgress(url string, fileName string) error {
	// get the content length of the file
	resp, err := d.http.Head(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	// create the progress bar
	bar := progressbar.DefaultBytes(
		int64(size),
		"downloading",
	)

	// download the file and update the progress bar
	resp, err = d.http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// use io.Copy to write to file and update the progress bar
	_, err = io.Copy(io.MultiWriter(file, bar), resp.Body)
	return err
}
