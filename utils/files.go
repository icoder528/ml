package utils

import (
	"bufio"
	"os"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

//ProceedLine 按行处理
func ProceedLine(path string, enc encoding.Encoding, action func(line []byte)) {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	for sc := bufio.NewScanner(getReader(file, enc)); sc.Scan(); {
		action(sc.Bytes())
	}
}

func getReader(file *os.File, enc encoding.Encoding) (reader *bufio.Reader) {

	if enc != nil {
		reader = bufio.NewReader(transform.NewReader(file, enc.NewDecoder()))
	} else {
		reader = bufio.NewReader(file)
	}
	return
}
