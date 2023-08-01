package utils

import (
	"bytes"
	"compress/zlib"
	"io/ioutil"
)

// ZipDecompression 解压
func ZipDecompression(d []byte) ([]byte, error) {
	deComperAfterBuffer, err := zlib.NewReader(bytes.NewBuffer(d))
	if err != nil {
		return nil, err
	}

	deComperAfterData, err := ioutil.ReadAll(deComperAfterBuffer)
	if err != nil {
		return nil, err
	}
	defer deComperAfterBuffer.Close()
	return deComperAfterData, nil
}
