package remote

import (
    "crypto/tls"
	"net"
    "net/http"
    "net/http/httputil"
)

type HttpConn struct {
    safeLev int
    clientConn   *httputil.ClientConn
    addr         string
}

/**
 * 构造函数
 */
func NewHttpConn(addr string, safeLev int) (*HttpConn, error) {
    ret := &HttpConn {
        safeLev: safeLev,
        addr: addr,
    }
    err := ret.connect()
    return ret, err
}

func (this *HttpConn) connect() error {
    conn, err := net.Dial("tcp", this.addr)
    if nil != err {
        return err
    }

    safeLev := this.safeLev
    if 0 < safeLev {
        conn = tls.Client(conn, &tls.Config{InsecureSkipVerify: 1 == safeLev})
    }

    this.clientConn = httputil.NewClientConn(conn, nil)
    return nil
}

func (this *HttpConn) Reset() error {
    this.Close()
    return this.connect()
}

func (this *HttpConn) Send(req *http.Request) (*http.Response, error) {
    return this.clientConn.Do(req)
}

func (this *HttpConn) Close() {
    this.clientConn.Close()
}
