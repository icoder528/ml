package utils

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sync"
)

//MemZip zip加载到内存中
type MemZip struct {
	fpath string
	files map[string][]byte

	sync.Mutex
}

//OpenMemZip 将zip文件加载到内存中
func OpenMemZip(fp string) (*MemZip, error) {
	fpath, err := filepath.Abs(fp)
	if err != nil {
		return nil, err
	}
	rc, err := zip.OpenReader(fpath)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	mz := &MemZip{fpath: fp, files: map[string][]byte{}}
	for _, f := range rc.File {
		if f.FileInfo().IsDir() {
			continue
		}
		subrc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer subrc.Close()

		bs, err := ioutil.ReadAll(subrc)
		if err != nil {
			return nil, err
		}
		mz.files[f.Name] = bs
	}

	return mz, nil
}

//Get 获取Zip中的文件
func (mz *MemZip) Get(fpath string) (r io.Reader, err error) {
	if bs, ok := mz.files[fpath]; ok {
		mz.Lock()
		r = bytes.NewReader(bs)
		mz.Unlock()
	} else {
		err = fmt.Errorf("%s not found file %s", mz.fpath, fpath)
	}
	return
}
