package myio

import (
    "io"
)

type WriteAtCloser interface {
    io.WriterAt
    io.WriteCloser
}

type WriteFromStream interface {
    SendFileAt(rs io.ReadCloser, w_off int64) error
}
