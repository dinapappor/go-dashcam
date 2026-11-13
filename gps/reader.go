package gps

import (
	"bufio"
	"io"
)

type paddingFilter struct {
	r    io.Reader
	prev byte
}

func (f *paddingFilter) Read(p []byte) (int, error) {
	n, err := f.r.Read(p)
	kept := 0
	for i := 0; i < n; i++ {
		b := p[i]
		if b == '\n' && f.prev != '\r' {
			continue
		}
		p[kept] = b
		kept++
		f.prev = b
	}

	return kept, err
}

func ReadSentences(r io.Reader, handle func(Sentence)) error {
	f := &paddingFilter{r: r}

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		sentence, err := ParseNMEA(scanner.Text())
		if err != nil {
			continue
		}
		handle(sentence)
	}
	return scanner.Err()
}
