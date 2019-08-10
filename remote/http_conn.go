package remote

import (
    "crypto/tls"
	"net"
    "net/http"
    "net/http/httputil"
)

type HttpConn struct {
    safeLev    int
    addr       string
    serverName string
    clientConn *httputil.ClientConn
}

/**
 * 构造函数
 */
func NewHttpConn(serverName string, safeLev int) (*HttpConn, error) {
    host, port, err := net.SplitHostPort(serverName)
    if nil != err {
        host = serverName
        port = "80"
        if 1 == safeLev {
            port = "443"
        }
    }
    ips, err := net.LookupIP(host)
    if nil != err {
        return nil, err
    }
    ret := &HttpConn {
        serverName: serverName,
        addr:       ips[0].String() + ":" + port,
        safeLev:    safeLev,
        clientConn: nil,
    }
    return ret, nil
}

func (this *HttpConn) Clone() *HttpConn {
    return &HttpConn {
        serverName: this.serverName,
        addr:       this.addr,
        safeLev:    this.safeLev,
        clientConn: nil,
    }
}

func (this *HttpConn) connect() error {
    conn, err := net.Dial("tcp", this.addr)
    if nil != err {
        return err
    }

    if 0 < this.safeLev {
        conn = tls.Client(conn, &tls.Config{
            InsecureSkipVerify: 1 == this.safeLev,
            ServerName: this.serverName,
        })
    }

    this.clientConn = httputil.NewClientConn(conn, nil)
    return nil
}

func (this *HttpConn) Connect(valid bool) error {
    _safeLev := this.safeLev
    if 0 < _safeLev {
        _safeLev = 1
        if valid {
            _safeLev++
        }
        this.safeLev = _safeLev
    }
    return this.connect()
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
