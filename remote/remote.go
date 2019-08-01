package remote

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

type HttpResource struct {
    parallelable  bool
    use_tls       int
    size          int64
	url_raw       string
    addr          string
    file_name     string
}

type HttpReader struct {
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
func New(url_raw string) (*HttpResource, error) {
	var err error
	for {
		uri, err := url.Parse(url_raw)
		if nil != err {
			break
		}

        use_tls := 0
        if "https" == uri.Scheme {
            use_tls = 1
        }
		host, port, err := net.SplitHostPort(uri.Host)
		if nil != err {
			host = uri.Host
			port = "80"
			if 1 == use_tls {
				port = "443"
			}
		}
		ips, err := net.LookupIP(host)
		if nil != err {
			break
		}
		this := &HttpResource{
			url_raw: url_raw,
			addr:    ips[0].String() + ":" + port,
			use_tls:  use_tls,
		}
		return this, nil
	}
	return nil, err
}

/**
 * 试着获取远端信息，文件名和内容长度
 * @return {error}
 */
func (this *HttpResource) GetMeta() error {
    cli, err := this.NewHttpReader()
    if nil != err {
		return err
    }
    err = cli.Conn(true)
	if nil != err {
		return err
    }
    req := cli.req
    req.Method = "HEAD"
    req.Header.Set("Connection", "Close")
    resp, err := cli.Send(req)
    resp.Body.Close()
    cli.Close()
	// 应答错误
	if 200 != resp.StatusCode {
		return errors.New(resp.Status)
	}

	acceptRanges := resp.Header.Get("Accept-Ranges")
	contentLength := resp.Header.Get("Content-Length")
    file_name := resp.Header.Get("Content-Disposition")

	this.parallelable = "" != acceptRanges && "none" != acceptRanges
	this.size, err = strconv.ParseInt(contentLength, 10, 64)
	if nil != err {
		return err
    }

    // 优先使用应答头里的文件名
    if 0 < len(file_name) {
        foo := strings.Split(file_name, "filename=")
        if 0 < len(foo) {
            file_name = foo[1]
        }
        if '"' == file_name[0] {
            file_name = file_name[1 : len(file_name) - 1]
        }
    }
    // 使用url的文件名
    if len(file_name) < 1 {
        file_name = path.Base(req.URL.Path)
    }
    this.file_name = file_name

	return nil
}

func (this *HttpResource) Filename() string {
    return this.file_name
}

func (this *HttpResource) Size() int64 {
    return this.size
}

func (this *HttpResource) Parallelable() bool {
    return this.parallelable
}

func (this *HttpResource) NewHttpReader() (*HttpReader, error) {
    url_raw := this.url_raw
    req, err := http.NewRequest("GET", url_raw, nil)
    if nil != err {
        return nil, err
    }
    req.Header = http.Header{}
    req.Header.Add("Connection", "keep-alive")

    return &HttpReader{
        req:     req,
        headers: &req.Header,
        url_raw: url_raw,
        addr:    this.url_raw,
        safeLev: this.use_tls,
    }, nil
}

func (this *HttpReader) Conn(valid bool) error {
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

func (this *HttpReader) Read(start int64, end int64, repeat int) (*Resp_t, error) {
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
