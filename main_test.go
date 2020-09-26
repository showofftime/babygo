package main

import (
	"testing"
)

func Test_scannerScanString(t *testing.T) {
	raw := "helloworld"
	
	src := convertStringToUint8Slice(raw)
	t.Logf("src: %v", src)
	scannerInit(src)

	str := scannerScanString()
	t.Logf(str)
}

func convertStringToUint8Slice(s string) []uint8 {
	src := []uint8{}
	for _, c := range s {
		src = append(src, uint8(c))
	}
	return src
}