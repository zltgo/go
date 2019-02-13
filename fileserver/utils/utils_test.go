package utils

import (
	"crypto/aes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"sync"
	"testing"
)

func Test_MD5(t *testing.T) {
	t.Log(MD5(""))
}

func Test_Sha1(t *testing.T) {
	t.Log(Sha1(""))
}

func Test_RandBytesMod(t *testing.T) {
	t.Log(RandBytesMod(16, 3))
}

func Test_RandStr(t *testing.T) {
	t.Log(RandStr(100))
}

func Test_IPv4(t *testing.T) {
	i, err := IPv4("192.168.1.1:9999")
	t.Log(i, err)
	if err != nil {
		t.Error(err)
	}
	if i != 3232235777 {
		t.Error("不会吧")
	}
}

func Test_Crypt(t *testing.T) {
	blockkey := RandBytes(16)
	block, _ := aes.NewCipher(blockkey)
	b := Encrypt(block, []byte("zyx"))

	c, err := Decrypt(block, b)
	if err != nil {
		t.Error(err)
		return
	}

	if string(c) != "zyx" {
		t.Error("not zyx")
	}

	return
}

func Test_Crypt1(t *testing.T) {
	blockkey := RandBytes(16)
	block, _ := aes.NewCipher(blockkey)
	b := Encrypt(block, []byte(""))
	t.Log(base64.URLEncoding.EncodeToString(b))
	c, err := Decrypt(block, b)
	if err != nil {
		t.Error(err)
		return
	}

	if string(c) != "" {
		t.Error("error")
	}

	return
}

func Benchmark_Hash32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Hash32("fdasfasdfdsafdsafasfsdafadsfdasfferfes")
	}
}

func Benchmark_Hash64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Hash32("fdasfasdfdsafdsafasfsdafadsfdasfferfes")
	}
}

func Benchmark_RandStr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStr(32)
	}
}

func Benchmark_RandBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandBytes(32)
	}
}

func Benchmark_Md5(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MD5("fdasfasdfdsafdsafasfsdafadsfdasfferfes")
	}
}

func Benchmark_Sha1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Sha1("fdasfasdfdsafdsafasfsdafadsfdasfferfes")
	}
}

func Benchmark_Encrypt(b *testing.B) {
	blockkey := RandBytes(16)
	block, _ := aes.NewCipher(blockkey)
	for i := 0; i < b.N; i++ {
		Encrypt(block, []byte("fdasfasdfdsafdsafasfsdafadsfdasfferfes"))
	}
}

func Benchmark_Block(b *testing.B) {
	blockkey := RandBytes(16)
	for i := 0; i < b.N; i++ {
		aes.NewCipher(blockkey)
	}
}

type UserInfo_noTag struct {
	m_tmp1     int //酱油1
	M_tmp2     int //酱油2
	M_username string
	M_city     string
	M_age      int
	M_cash     float64
	M_sex      bool
	M_zip      []byte
}

func Benchmark_Gob(b *testing.B) {
	x := UserInfo_noTag{}
	gob.Register(x)
	for i := 0; i < b.N; i++ {
		c, err := EncodeGob(x)
		if err != nil {
			b.Error(err)
			break
		}
		err = DecodeGob(c, &x)
		if err != nil {
			b.Error(err)
			break
		}
	}
}

func Benchmark_Json(b *testing.B) {
	x := UserInfo_noTag{}

	for i := 0; i < b.N; i++ {
		c, err := json.Marshal(x)
		if err != nil {
			b.Error(err)
			break
		}
		err = json.Unmarshal(c, &x)
		if err != nil {
			b.Error(err)
			break
		}
	}
}

func fun1(lock *sync.Mutex) {
	lock.Lock()
	lock.Unlock()
}

func fun3(lock *sync.RWMutex) {
	lock.Lock()
	lock.Unlock()
}

func fun4(lock *sync.RWMutex) {
	lock.RLock()
	lock.RUnlock()
}

func Benchmark_Lock(b *testing.B) {
	var lock sync.Mutex
	for i := 0; i < b.N; i++ {
		fun1(&lock)
	}
}

func Benchmark_WLock(b *testing.B) {
	var lock sync.RWMutex
	for i := 0; i < b.N; i++ {
		fun3(&lock)
	}
}

func Benchmark_RLockR(b *testing.B) {
	var lock sync.RWMutex
	for i := 0; i < b.N; i++ {
		fun4(&lock)
	}
}

func fun2(lock *sync.Mutex) {
	lock.Lock()
	defer lock.Unlock()
}
func Benchmark_DeferLock(b *testing.B) {
	var lock sync.Mutex
	for i := 0; i < b.N; i++ {
		fun2(&lock)
	}
}
