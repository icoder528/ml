package utils

import (
	"bufio"
	"io"
	"strings"
)

//TravelLines 按行遍历
func TravelLines(r io.Reader, step string, fn func(string, []string)) error {
	br := bufio.NewReader(r)
	for {
		line, err := readline(br)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if strings.HasPrefix(line, "#") {
			continue
		}
		fn(line, strings.Split(strings.TrimSpace(string(line)), step))
	}
	return nil
}

func readline(r *bufio.Reader) (string, error) {
	var (
		isPrefix = true
		err      error
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}
