package captcha

import (
	"testing"

	. "github.com/zltgo/fileserver/utils"
)

//测试Verify
func Test_Verify(t *testing.T) {
	digits := RandBytesMod(6, byte(len(c_strFonts)))

	id := Digits2Id(digits)
	t.Log(len(id), id)

	chars := make([]byte, 6)
	for i := 0; i < 6; i++ {
		chars[i] = c_strFonts[digits[i]]
	}
	str := string(chars)
	t.Log(str)

	if Verify(id, str) != true {
		t.Error("Verify error")
	}
}

//测试验证码生产效率
func Benchmark_New(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewStd()
	}
}

func Benchmark_Image(b *testing.B) {
	id := NewStd()
	for i := 0; i < b.N; i++ {
		_, err := GetImage(id, 240, 80)
		if err != nil {
			b.Error(err)
			break
		}
	}
}
