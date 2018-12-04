// +build gofuzz

package fex

func Fuzz(b []byte) (rc int) {
	_, _ = CompileExtractor(string(b))
	return 0
}
