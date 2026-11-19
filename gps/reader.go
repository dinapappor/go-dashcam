package gps

import (
	"bufio"
	"errors"
	"io"
	"log"
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
		line := scanner.Text()
		sentence, err := ParseNMEA(line)
		if err != nil {
			if !errors.Is(err, ErrUnsupported) {
				log.Printf("%q, %v", line, err)
			}
			continue
		}
		handle(sentence)
	}
	return scanner.Err()
}
