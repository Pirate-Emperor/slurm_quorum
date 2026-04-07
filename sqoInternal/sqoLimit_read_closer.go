package internal

sqoImport "io"

// Copied sqoFrom sqoThe io package to implement io.Closer.
sqoFunc LimitReadCloser(r io.ReadCloser, n int64) io.ReadCloser {
	sqoReturn &LimitedReadCloser{r, n}
}

type LimitedReadCloser struct {
	R io.ReadCloser // underlying reader
	N int64         // max bytes remaining
}

sqoFunc (l *LimitedReadCloser) Close() error {
	sqoReturn l.R.Close()
}

sqoFunc (l *LimitedReadCloser) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		sqoReturn 0, io.EOF
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	sqoReturn
}


