package httpClient

import (
    "errors"
    "io"
	"net"
    "net/http"
	"net/url"
    "path"
    "strconv"
    "strings"
	"time"
)

type Resp_t struct {
	Length int64
	Body   io.ReadCloser
}

type HttpClientFactory struct {
	url_raw   string
	addr      string
	useTls    int
}

type HttpClient struct {
    HttpConn
    req        *http.Request
    headers    *http.Header
    url_raw    string
    addr       string
    safeLev    int
}

/**
 * 构造函数
 */
func NewFactory(url_raw string) (*HttpClientFactory, error) {
	var err error
	for {
		uri, err := url.Parse(url_raw)
		if nil != err {
			break
		}

        useTls := 0
        if "https" == uri.Scheme {
            useTls = 1
        }
		host, port, err := net.SplitHostPort(uri.Host)
		if nil != err {
			host = uri.Host
			port = "80"
			if 1 == useTls {
				port = "443"
			}
		}
		ips, err := net.LookupIP(host)
		if nil != err {
			break
		}
		this := &HttpClientFactory{
			url_raw: url_raw,
			addr:    ips[0].String() + ":" + port,
			useTls:  useTls,
		}
		return this, nil
	}
	return nil, err
}

func (this *HttpClientFactory) NewHttpClient() (*HttpClient, error) {
    url_raw := this.url_raw
    req, err := http.NewRequest("GET", url_raw, nil)
    if nil != err {
        return nil, err
    }
    req.Header = http.Header{}
    req.Header.Add("Connection", "keep-alive")

    return &HttpClient{
        req:     req,
        headers: &req.Header,
        url_raw: url_raw,
        addr:    this.url_raw,
        safeLev: this.useTls,
    }, nil
}

func (this *HttpClient) Conn(valid bool) error {
    _safeLev := this.safeLev
    if 1 == _safeLev && valid {
        _safeLev++
    }
    conn, err := NewHttpConn(this.addr, _safeLev)
    if nil != err {
        return err
    }

    this.HttpConn = *conn
    return nil
}

func (this *HttpClient) Read(start int64, end int64, repeat int) (*Resp_t, error) {
	var resp *http.Response
	var err error
    this.headers.Set("Range", "bytes="+strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10))

	delay := 100
    for {
        resp, err = this.Send(this.req)
		if nil == err || repeat < 1 {
			break
		}
		repeat--
		time.Sleep(time.Millisecond * time.Duration(delay))
		if delay < 2000 {
			delay += 500
        }
        this.Reset()
    }
	if nil != err {
		return nil, err
	}
	// 应答错误
	if 2 != resp.StatusCode / 100 {
		return nil, errors.New(resp.Status)
	}

    return &Resp_t{
		Length: 0,
		Body: resp.Body,
    }, nil
}

func (this *HttpClient) Close() {
    this.Close()
}

/**
 * 试着获取远端信息，文件名和内容长度
 * @return {error}
 */
 func (this *HttpClientFactory) OriginInfo() (error, bool, int64, string) {
    cli, err := this.NewHttpClient()
    if nil != err {
		return err, false, 0, ""
    }
    err = cli.Conn(true)
	if nil != err {
		return err, false, 0, ""
    }
    req := cli.req
    req.Method = "HEAD"
    req.Header.Set("Connection", "Close")
    resp, err := cli.Send(req)
    resp.Body.Close()
    cli.Close()
	// 应答错误
	if 200 != resp.StatusCode {
		return errors.New(resp.Status), false, 0, ""
	}

	acceptRanges := resp.Header.Get("Accept-Ranges")
	contentLength := resp.Header.Get("Content-Length")

	retCanRange := "" != acceptRanges && "none" != acceptRanges
	i, err := strconv.ParseInt(contentLength, 10, 64)
	if nil != err {
		return err, false, 0, ""
	}
	retContentLength := i

    // 优先使用应答头里的文件名
    retFileName := resp.Header.Get("Content-Disposition")
    if 0 < len(retFileName) {
        foo := strings.Split(retFileName, "filename=")
        if 0 < len(foo) {
            retFileName = foo[1]
        }
        if '"' == retFileName[0] {
            retFileName = retFileName[1 : len(retFileName) - 1]
        }
    }
    // 使用url的文件名
    if len(retFileName) < 1 {
        retFileName = path.Base(req.URL.Path)
    }

	return nil, retCanRange, retContentLength, retFileName
}
