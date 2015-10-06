package sx

import "io"
import "bytes"
import "encoding/base64"

type writeDecoder struct {
	sink   io.Writer
	b64dec io.Reader
	b64sr  switchableReader
	err    error
}

func newWriteDecoder(sink io.Writer) *writeDecoder {
	wd := &writeDecoder{sink: sink}
	wd.b64dec = base64.NewDecoder(base64.StdEncoding, &filteringReader{&wd.b64sr})
	return wd
}

func (wd *writeDecoder) Write(b []byte) (int, error) {
	if wd.err != nil {
		return 0, wd.err
	}

	wd.b64sr.Reader = bytes.NewReader(b)
	n, err := io.Copy(wd.sink, wd.b64dec)
	if err != nil {
		wd.err = err
		return int(n), err
	}

	return len(b), nil
}

type switchableReader struct {
	Reader io.Reader
}

func (sr *switchableReader) Read(b []byte) (int, error) {
	if sr.Reader == nil {
		return 0, io.EOF
	}

	return sr.Reader.Read(b)
}

type filteringReader struct {
	wrapped io.Reader
}

func (r *filteringReader) Read(p []byte) (int, error) {
	n, err := r.wrapped.Read(p)
	for n > 0 {
		offset := 0
		for i, b := range p[:n] {
			if b != '\r' && b != '\n' && b != ' ' && b != '\t' {
				if i != offset {
					p[offset] = b
				}
				offset++
			}
		}
		if offset > 0 {
			return offset, err
		}
		// Previous buffer entirely whitespace, read again
		n, err = r.wrapped.Read(p)
	}
	return n, err
}
