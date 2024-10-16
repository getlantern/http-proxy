package water

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
)

//go:generate mockgen -destination=mock_downloader.go -package=water . WASMDownloader

type WASMDownloader interface {
	DownloadWASM(context.Context, io.Writer) error
}

type downloader struct {
	urls           []string
	httpClient     *http.Client
	httpDownloader WASMDownloader
}

type DownloaderOption func(*downloader)

// NewWASMDownloader creates a new WASMDownloader instance.
func NewWASMDownloader(urls []string, client *http.Client) (WASMDownloader, error) {
	if len(urls) == 0 {
		return nil, log.Error("WASM downloader requires URLs to download but received empty list")
	}
	return &downloader{
		urls:       urls,
		httpClient: client,
	}, nil
}

// DownloadWASM downloads the WASM file from the given URLs, verifies the hash
// sum and writes the file to the given writer.
func (d *downloader) DownloadWASM(ctx context.Context, w io.Writer) error {
	joinedErrs := errors.New("failed to download WASM from all URLs")
	for _, url := range d.urls {
		if strings.HasPrefix(url, "magnet:?") {
			// Skip magnet links for now
			joinedErrs = errors.Join(joinedErrs, errors.New("magnet links are not supported"))
			continue
		}
		tempBuffer := &bytes.Buffer{}
		err := d.downloadWASM(ctx, tempBuffer, url)
		if err != nil {
			joinedErrs = errors.Join(joinedErrs, err)
			continue
		}

		_, err = tempBuffer.WriteTo(w)
		if err != nil {
			joinedErrs = errors.Join(joinedErrs, err)
			continue
		}

		return nil
	}
	return joinedErrs
}

// downloadWASM checks what kind of URL was given and downloads the WASM file
// from the URL. It can be a HTTPS URL or a magnet link.
func (d *downloader) downloadWASM(ctx context.Context, w io.Writer, url string) error {
	switch {
	case strings.HasPrefix(url, "http://"), strings.HasPrefix(url, "https://"):
		if d.httpDownloader == nil {
			d.httpDownloader = NewHTTPSDownloader(d.httpClient, url)
		}
		return d.httpDownloader.DownloadWASM(ctx, w)
	default:
		return log.Errorf("unsupported protocol: %s", url)
	}
}
