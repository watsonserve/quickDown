package remote

import (
    "errors"
    "io"
    "net/http"
    "strconv"
    "time"
)

type Resp_t struct {
	Length int64
	Body   io.ReadCloser
}

type HttpReader struct {
    conn       *HttpConn
    req        *http.Request
    headers    *http.Header
}

func (this *HttpResource) NewHttpReader() (*HttpReader, error) {
    req, err := http.NewRequest("GET", this.rawUrl, nil)
    if nil != err {
        return nil, err
    }
    req.Header = http.Header{}
    req.Header.Add("Connection", "keep-alive")

    return &HttpReader{
        conn:       this.connector.Clone(),
        req:        req,
        headers:    &req.Header,
    }, nil
}

func (this *HttpReader) Conn(valid bool) error {
    return this.conn.Connect(valid)
}

func (this *HttpReader) Read(start int64, end int64, repeat int) (*Resp_t, error) {
	var resp *http.Response
	var err error
    this.headers.Set("Range", "bytes=" + strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10))

	delay := 100
    for {
        resp, err = this.conn.Send(this.req)
		if nil == err || repeat < 1 {
			break
		}
		repeat--
		time.Sleep(time.Millisecond * time.Duration(delay))
		if delay < 2000 {
			delay += 500
        }
        this.conn.Reset()
    }
	if nil != err {
		return nil, err
	}
	// 应答错误
	if 2 != resp.StatusCode / 100 {
		return nil, errors.New(resp.Status)
	}

    return &Resp_t {
		Length: 0,
		Body: resp.Body,
    }, nil
}

func (this *HttpReader) Reset() error {
    return this.conn.Reset()
}

func (this *HttpReader) Send(req *http.Request) (*http.Response, error) {
    return this.conn.Send(req)
}

func (this *HttpReader) Close() {
    this.conn.Close()
}