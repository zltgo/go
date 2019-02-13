/***********************************************************************************\
盘点那些踩过的坑
	1）作为返回值，最好不要写成(err error)，或者在函数开始声明var err error,因为函数内类似的调用
	db, err := sql.Open()会重新声明一个err，有可能导致返回值错乱
	2）go语言只有四种类型map、chan、slice、interface{}都属于引用类型（默认值均为nil），内部维护了对象的指针，对所
	指向对象的更改会影响到对象本身和所有指向该对象的引用类型；要更改引用对象本身，在函数传递时必须
	传递引用类型本身的指针，类似于理解const char* const ptr；map使用前必须先使用make(map)赋值,否则会panic
	3）类的成员函数如：func (ds *DataSet) OpenDb() (*sql.DB, error) ；func (tb table) key2If(v reflect.Value) []interface{} ，
	一般来讲，对类的属性有更改使用指针形式，否则使用值的形式；指针形式兼容值的形式，需要时系统会自动生成值形式
	的同名函数，但反过来则不会。
	4）函数有多个返回值时，例如 arg := range args，只返回第一个值，并且编译和运行都不报错，其实想要实现的效果
	可能是 _,arg := range args
	5）sql.DB内部实现了连接池，全局有一个db sql.DB即可
	6）拥有同一个 包名的多个文件均可以有默认init函数，执行顺序以文件名为准
	7）slice为引用类型，只要操作没有重新分配内存（即cap变大）,操作会影响所有指向该区域的对象，例如：
		x := []int{1, 2, 3}
		y := x[:2]
		y = append(y, 4)
		y[1] = 5
		Assert(x == [1 5 4])
	8）map比slice更彻底，引用对象之间的任何操作都会影响到别人，如要复制，使用copy()
	9）双引号字符串与反引号字符串，有可能看起来一样，实际不相等，因为反引号中没有对字符进行转义，例如：
		s1 := "\\b"
		s2 := `\\b`
		s3 := `\b`
		Assert(s1 != s2 && s1 == s3)
	10) 用range遍历slice时会赋值slice中的元素，修改不会保存到slice中，要修改则直接使用slice[i]
	11) 可以对空的slice 、map进行len(nil), range nil操作， 不用担心会panic
	12) bson在decode到map中时，能保留数据的类型信息（int, int64, float64等）， 而json只能保存为float64;
	      json再把float64转换成int时, 可能会产生精度损失, 得用json.Number;
	      bson会把结构体字段名改为小写， 而json不会;
	      bson对time.Time能够decode到map中，而json在decode之后的类型为string
	13) 不可以对interfase{}(nil)进行类型断言操作，会panic, 但是能放在switch 中进行断言：
		var iii interface{}
		switch iii.(type) {
		case int:
			fmt.Println("int")
		case nil:
			fmt.Println("nil")
		default:
			fmt.Println("unknown")
		}
	14) reflect.TypeOf(http.ResponseWriter(nil)) 的结果为nil， 而不是http.ResponseWriter, 解决方法参见
	      inject.InterfaceOf((*http.ResponseWriter)(nil)).
	15) string 和 []byte互相转换，不怕byte数组里面有0,\0
	16) map的for range 遍历是随即的，例如:
		mp:= map[int]bool {
			1:true,
			2:true,
			3:true,
		}
		for num, _ := range mp {
			fmt.Print(num)
		}
		会打印出312、321、132等，每次都不一样
	17) defer 与 c++中类的析构用法有不同之处，类有作用域的概念，而defer总是在整个的作用域中，离开函数时才被调用
	      例如下面的函数，只有在整个函数退出时，defer才会被调用，而不是在每一次for循环的迭代过程中。尽量不在循环体内部
		调用defer，如有需要，可以用函数包裹起来迭代执行。
		func(){
			for i:=0; i< 4; i++ {
				var mu sync.Mutex
				mu.Lock
				defer mu.Unlock()
			}
		}

		func(){
			for i:=0; i<100; i++ {
				func () {
					var mu sync.Mutex
					mu.Lock
					defer mu.Unlock()
				}()
			}
		}
	18) var i = 0; for i = range len(slice){...}  这段代码执行后 i = len(silce) - 1.
	19) len(strings.Split("", ",")) == 1
	20) json 转换成map[string]interface{}时并不能准确保留原来的类型信息，例如数字都会转成float64, []string会转换
	     成[]interface{}
	21) swith v := vv.(type) 该操作与range操作一样，会生成一个vv的副本拷贝
	22）interface{}只有当类型和值均为nil时， 与nil进行比较判断才会返回ture，参见如下代码：
		func test1(i int) *ABC {
			if i != 0 {
				return &ABC{}
			}
			return nil
		}
		func test2(i int) interface{} {
			return test1(i)
		}
		func main() {
			v := test2(0)
			//<nil> false *main.ABC
			fmt.Println(v, v == nil, reflect.TypeOf(v))
		}
\***********************************************************************************/
package utils

import (
	"bytes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

//检查err是否为nil，不为nil则panic
func CheckErr(err error) {
	if err != nil {
		panic(err)
	}
	return
}

//断言expression为真，否则panic
func Assert(expression bool, format string, a ...interface{}) {
	if !expression {
		panic(fmt.Sprintf(format, a...))
	}
}

//获取指针或者值的值类型
func Any2Value(arg interface{}) reflect.Value {
	return reflect.Indirect(reflect.ValueOf(arg))
}

//获取指针所指向的值类型
func Ptr2Value(ptr interface{}) reflect.Value {
	return reflect.ValueOf(ptr).Elem()
}

//获取指针或值的地址接口
func Value2IfAddr(v reflect.Value) interface{} {
	if v.Type().Kind() == reflect.Ptr {
		return v.Interface()
	}
	return v.Addr().Interface()
}

//获取值或指针的真实类型
func IndirectType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}

//计算MD5值，结果以十六进制字符串表示，共32个字符
func MD5(input string) string {
	t := md5.New()
	io.WriteString(t, input)
	return hex.EncodeToString(t.Sum(nil))
}

//哈西算法，结果以十六进制字符串表示，共40个字符
func Sha1(input string) string {
	t := sha1.New()
	io.WriteString(t, input)
	return hex.EncodeToString(t.Sum(nil))
}

//将字符串转换成任意的基础类型的reflect.Value
func Str2Value(s string, t reflect.Type) (v reflect.Value, err error) {
	switch t.Kind() {
	case reflect.String:
		v = reflect.ValueOf(s)
	case reflect.Int:
		var i int64
		i, err = strconv.ParseInt(s, 0, 0)
		if err == nil {
			v = reflect.ValueOf(int(i))
		}
	case reflect.Uint:
		var n uint64
		n, err = strconv.ParseUint(s, 0, 0)
		if err == nil {
			v = reflect.ValueOf(uint(n))
		}
	case reflect.Ptr:
		if t.Elem().String() == "regexp.Regexp" {
			var rx *regexp.Regexp
			rx, err = regexp.Compile(s)
			if err == nil {
				v = reflect.ValueOf(rx)
			}
		} else {
			err = fmt.Errorf("不支持的数据类型：%s", t.String())
		}
	case reflect.Bool:
		var b bool
		b, err = strconv.ParseBool(s)
		if err == nil {
			v = reflect.ValueOf(b)
		}
	case reflect.Float32:
		var f float64
		f, err = strconv.ParseFloat(s, 32)
		if err == nil {
			v = reflect.ValueOf(float32(f))
		}
	case reflect.Float64:
		var f float64
		f, err = strconv.ParseFloat(s, 64)
		if err == nil {
			v = reflect.ValueOf(f)
		}
	case reflect.Int8:
		var i int64
		i, err = strconv.ParseInt(s, 0, 8)
		if err == nil {
			v = reflect.ValueOf(int8(i))
		}
	case reflect.Int16:
		var i int64
		i, err = strconv.ParseInt(s, 0, 16)
		if err == nil {
			v = reflect.ValueOf(int16(i))
		}
	case reflect.Int32:
		var i int64
		i, err = strconv.ParseInt(s, 0, 32)
		if err == nil {
			v = reflect.ValueOf(int32(i))
		}
	case reflect.Int64:
		var i int64
		i, err = strconv.ParseInt(s, 0, 64)
		if err == nil {
			v = reflect.ValueOf(i)
		}
	case reflect.Uint8:
		var n uint64
		n, err = strconv.ParseUint(s, 0, 8)
		if err == nil {
			v = reflect.ValueOf(uint8(n))
		}
	case reflect.Uint16:
		var n uint64
		n, err = strconv.ParseUint(s, 0, 16)
		if err == nil {
			v = reflect.ValueOf(uint16(n))
		}
	case reflect.Uint32:
		var n uint64
		n, err = strconv.ParseUint(s, 0, 32)
		if err == nil {
			v = reflect.ValueOf(uint32(n))
		}
	case reflect.Uint64:
		var n uint64
		n, err = strconv.ParseUint(s, 0, 64)
		if err == nil {
			v = reflect.ValueOf(n)
		}
	default:
		err = fmt.Errorf("不支持的数据类型：%s", t.String())
	}
	return
}

//将字符串转换成任意的基础类型
func Str2Basic(s string, ptr interface{}) error {
	v := Ptr2Value(ptr)
	tmp, err := Str2Value(s, v.Type())
	if err == nil {
		v.Set(tmp)
	}
	return err
}

//基础数据类型转为float64
func Basic2Float64(in interface{}) (rv float64, err error) {
	switch v := in.(type) {
	case string:
		rv, err = strconv.ParseFloat(v, 64)
	case int:
		rv = float64(v)
	case uint:
		rv = float64(v)
	case int8:
		rv = float64(v)
	case int16:
		rv = float64(v)
	case int32:
		rv = float64(v)
	case int64:
		rv = float64(v)
	case uint8:
		rv = float64(v)
	case uint16:
		rv = float64(v)
	case uint32:
		rv = float64(v)
	case uint64:
		rv = float64(v)
	case float32:
		rv = float64(v)
	case float64:
		rv = v
	default:
		err = fmt.Errorf("不支持的数据类型：%s", reflect.TypeOf(in).String())
	}
	return
}

//获取源字符串s的前size个子字符
func Substr(s string, size int) string {
	if size >= len(s) || size < 0 {
		return s
	}
	b := []byte(s)
	return string(b[:size])
}

// randomBytes returns a byte slice of the given size read from CSPRNG.
//4019ns/op，速度不快
func RandBytes(size int) (b []byte) {
	b = make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("RandBytes: error reading random source: " + err.Error())
	}
	return
}

// random string  a-zA-Z
//[97 122 65 90]
func RandStr(size int) string {
	b := RandBytesMod(size, 52)
	for i := 0; i < size; i++ {
		if b[i] < 26 {
			b[i] += 65
		} else {
			b[i] += 71
		}
	}
	return string(b)
}

// randomBytesMod returns a byte slice of the given size, where each byte is
// a random number modulo mod.
//mod为2时，只随机出0，1
func RandBytesMod(size int, mod byte) (b []byte) {
	if size == 0 {
		return nil
	}

	maxrb := 255 - byte(256%int(mod))
	b = make([]byte, size)
	i := 0
	for {
		r := RandBytes(size + (size / 4))
		for _, c := range r {
			if c > maxrb {
				// Skip this number to avoid modulo bias.
				continue
			}
			b[i] = c % mod
			i++
			if i == size {
				return
			}
		}
	}
}

//洗牌，返回size长度的数组，每一个对应唯一的序号[0, size -1]
func Shuffle(size int) (b []byte) {
	Assert(size < 256, "Shuffle: size argument is too big")
	if size == 0 {
		return nil
	}

	//初值按顺序排列
	b = make([]byte, size)
	for i := 0; i < len(b); i++ {
		b[i] = byte(i)
	}

	//随机两两交换
	c := RandBytesMod(2*size, byte(size))
	var tmp byte
	for i := 0; i < len(c); i += 2 {
		tmp = b[c[i]]
		b[c[i]] = b[c[i+1]]
		b[c[i+1]] = tmp
	}
	return
}

func Hash32(input string) uint32 {
	h := fnv.New32()
	io.WriteString(h, input)
	return h.Sum32()
}

func Hash64(input string) uint64 {
	h := fnv.New64()
	io.WriteString(h, input)
	return h.Sum64()
}

//获取hostport"127.0.0.1：8080"中的IP地址的值
func IPv4(hostport string) (int64, error) {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return 0, err
	}

	//[::1]:51831localhost:
	if strings.Contains(host, "::") {
		host = "127.0.0.1"
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return 0, fmt.Errorf("invalid IP: %s", host)
	}
	//return int64(binary.BigEndian.Uint32(ip.To4())), nil
	return strconv.ParseInt(hex.EncodeToString(ip.To4()), 16, 64)
}

// Encryption -----------------------------------------------------------------

// encrypt encrypts a value using the given block in counter mode.
//
// A random initialization vector (http://goo.gl/zF67k) with the length of the
// block size is prepended to the resulting ciphertext.
func Encrypt(block cipher.Block, value []byte) []byte {
	iv := RandBytes(block.BlockSize())
	t := fnv.New64()
	t.Write(value)
	t.Write(iv)
	value = t.Sum(value)

	// Encrypt it.
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(value, value)
	// Return iv + ciphertext.
	return append(iv, value...)
}

//加密并转换为base64.URL编码
func EncryptUrl(block cipher.Block, value []byte) string {
	return base64.URLEncoding.EncodeToString(Encrypt(block, value))
}

// decrypt decrypts a value using the given block in counter mode.
//
// The value to be decrypted must be prepended by a initialization vector
// (http://goo.gl/zF67k) with the length of the block size.
var ErrDecrypt error = errors.New("Decrypt: the value could not be decrypted")

func Decrypt(block cipher.Block, value []byte) ([]byte, error) {
	size := block.BlockSize()
	if len(value) < size+8 {
		return nil, ErrDecrypt
	}

	// Extract iv.
	iv := value[:size]
	// Extract ciphertext.
	value = value[size:]
	// Decrypt it.
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(value, value)

	//check
	rv := len(value) - 8
	t := fnv.New64()
	t.Write(value[:rv])
	t.Write(iv)
	h := t.Sum(nil)

	for i := 0; i < 8; i++ {
		if h[i] != value[rv+i] {
			return nil, ErrDecrypt
		}
	}
	return value[:rv], nil
}

//解密EncryptUrl加密的内容
func DecryptUrl(block cipher.Block, str string) ([]byte, error) {
	b, err := base64.URLEncoding.DecodeString(str)
	if err != nil {
		return nil, ErrDecrypt
	}
	return Decrypt(block, b)
}

//Gob编码，为什么比JSON慢10倍？
func EncodeGob(obj interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(obj)
	return buf.Bytes(), err
}

//不科学啊，84849 ns/op
func DecodeGob(encoded []byte, ptr interface{}) error {
	buf := bytes.NewBuffer(encoded)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(ptr)
	return err
}
