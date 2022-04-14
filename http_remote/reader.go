package http_remote

import (
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Resp_t struct {
	Length int64
	Body   io.ReadCloser
}

type HttpReader struct {
	client  *http.Client
	req     *http.Request
	headers *http.Header
}

func newHttpClient(uri *url.URL) *http.Client {
	var transport http.RoundTripper = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: "https" == uri.Scheme,
			ServerName:         uri.Host,
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport: transport,
	}
}

func newHttpReader(rawUrl string) (*HttpReader, error) {
	uri, err := url.Parse(rawUrl)
	if nil != err {
		return nil, err
	}

	req, err := http.NewRequest("GET", rawUrl, nil)
	if nil != err {
		return nil, err
	}

	req.Header = http.Header{}
	req.Header.Add("Connection", "keep-alive")

	return &HttpReader{
		client:  newHttpClient(uri),
		req:     req,
		headers: &req.Header,
	}, nil
}

func (this *HttpReader) read(start int64, end int64, repeat int) (*Resp_t, error) {
	var resp *http.Response
	var err error
	this.headers.Set("Range", "bytes="+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10))

	delay := 100
	for {
		resp, err = this.client.Do(this.req)
		if nil == err || repeat < 1 {
			break
		}
		repeat--
		time.Sleep(time.Millisecond * time.Duration(delay))
		if delay < 2000 {
			delay += 500
		}
	}
	if nil != err {
		return nil, err
	}
	// 应答错误
	if 2 != resp.StatusCode/100 {
		return nil, errors.New(resp.Status)
	}

	return &Resp_t{
		Length: 0,
		Body:   resp.Body,
	}, nil
}

func (this *HttpReader) Read(start int64, end int64, repeat int) (*Resp_t, error) {
	rsp, err := this.read(start, end, repeat)
	if nil != err {
		// rebuild
		rawUrl := this.req.URL.String()
		that, _err := newHttpReader(rawUrl)

		if nil == _err {
			this.client = that.client
			this.req = that.req
			this.headers = that.headers
			rsp, err = this.read(start, end, repeat)
		}
	}
	return rsp, err
}
