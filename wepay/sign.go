package wepay

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/zltgo/reflectx"
)

const (
	MD5        = "MD5"
	SHA1       = "SHA1"
	SHA256     = "SHA256"
	SHA512     = "SHA512"
	HMACSHA256 = "HMAC-SHA256"
	HMACSHA512 = "HMAC-SHA512"
)

// 微信支付中将对数据里面的内容进行鉴权，确定携带的信息是真实、有效、合理的。因
// 此，这里将定义生成sign 字符串的方法。
// a.对所有传入参数按照字段名的ASCII 码从小到大排序（字典序）后，使用URL 键值对的
// 格式（即key1=value1&key2=value2…）拼接成字符串string1，注意：值为空的参数不参与
// 签名，不包括sign字段本身；
// b. 在string1 最后拼接上key=Key( 商户支付密钥) 得到stringSignTemp 字符串， 并对
// stringSignTemp 进行md5 运算，再将得到的字符串所有字符转换为大写，得到sign值signValue。

type SignOpts struct {
	ApiKey   string
	HashKey  string   //used for hash func
	SignKey  string   `default:"sign"`
	OmitKeys []string //some keys needed to skip
	TagName  string   `default:"xml"`
	SignType string   `default:"MD5"` //SHA1,HMAC-SHA256,HMAC-SHA512
}

type Signer struct {
	mapper   *reflectx.Mapper
	omitKeys []string //some keys needed to skip
	signKey  string
	apiKey   string
	hashFunc func() hash.Hash
	hashKey  string
}

// Default tagName is "xml", default sign key is "sign".
// Default TagFunc is FieldNameToUnderscore.
//	type Example struct {
//		AppId string `xml:"appid"`
//		MchId string `xml:"mch_id"`
//		Sign string `xml:"sign"`
//        TotalFee      int // auto convert field name to "total_fee"
//	}
func NewSigner(opts SignOpts) *Signer {
	reflectx.SetDefault(&opts)
	s := &Signer{
		mapper:   reflectx.NewMapper(opts.TagName, reflectx.FieldNameToUnderscore),
		omitKeys: opts.OmitKeys,
		hashKey:  opts.HashKey,
		apiKey:   opts.ApiKey,
	}

	switch opts.SignType {
	case MD5:
		s.hashFunc = md5.New
	case SHA1:
		s.hashFunc = sha1.New
	case HMACSHA256, SHA256:
		s.hashFunc = sha256.New
	case HMACSHA512, SHA512:
		s.hashFunc = sha512.New
	default:
		panic("unknown sign type")
	}

	return s
}

// Sign gets the sign value of the struct.
func (s *Signer) Sign(ptr interface{}) string {
	val := reflect.ValueOf(ptr).Elem()
	typ := val.Type()
	structMap := s.mapper.TypeMap(typ)
	// only the first level fields are valid
	children := structMap.Tree.Children
	// sorted by key
	sort.Sort(byName(children))

	var kvString string
	var signField reflect.Value
	for _, fi := range children {
		// skip the keys provided by omitkeys.
		if s.skip(fi.Name) {
			continue
		}
		//find the field of sign
		if fi.Name == s.signKey {
			signField = reflectx.FieldByIndexes(val, fi.Index)
			continue
		}

		// skip invalid value or nil ptr.
		fV := reflectx.FieldByIndexesReadOnly(val, fi.Index)
		fV = reflect.Indirect(fV)
		if !fV.IsValid() {
			continue
		}

		//convert value to string
		vStr, err := reflectx.ValueToStr(fV)
		if err != nil {
			panic(err)
		}
		//("bar=baz&foo=quux") sorted by key, omitempty.
		if vStr != "" {
			kvString = kvString + fi.Name + "=" + vStr + "&"
		}
	}

	//add "key=API_KEY"
	if s.apiKey != "" {
		kvString = kvString + "key=" + s.apiKey
	} else {
		kvString = strings.TrimSuffix(kvString, "&")
	}

	//hash, MD5,SHA1,SHA256,SHA512
	//must create here for thread-safe
	hash := s.hashFunc()
	if s.hashKey != "" {
		// hmac(New, nil) is diffrent from New.
		hash = hmac.New(s.hashFunc, []byte(s.hashKey))
	}
	io.WriteString(hash, kvString)
	signStr := hex.EncodeToString(hash.Sum(nil))
	signStr = strings.ToUpper(signStr)
	signField.SetString(signStr)
	return signStr
}

//Verify checks the signature of obj.
//It returns true if the signature is correct.
func (s *Signer) Verify(obj interface{}) bool {
	typ, val := reflectx.Indirect(obj)
	structMap := s.mapper.TypeMap(typ)
	// only the first level fields are valid
	children := structMap.Tree.Children
	// sorted by key
	sort.Sort(byName(children))

	var kvString, signature string
	for _, fi := range children {
		// skip the keys provided by omitkeys.
		if s.skip(fi.Name) {
			continue
		}

		// skip invalid value or nil ptr.
		fV := reflectx.FieldByIndexesReadOnly(val, fi.Index)
		fV = reflect.Indirect(fV)
		if !fV.IsValid() {
			continue
		}

		//find the field of sign
		if fi.Name == s.signKey {
			signature = fV.String()
			continue
		}

		//convert value to string
		vStr, err := reflectx.ValueToStr(fV)
		if err != nil {
			panic(err)
		}
		//("bar=baz&foo=quux") sorted by key, omitempty.
		if vStr != "" {
			kvString = kvString + fi.Name + "=" + vStr + "&"
		}
	}

	//add "key=API_KEY"
	if s.apiKey != "" {
		kvString = kvString + "key=" + s.apiKey
	} else {
		kvString = strings.TrimSuffix(kvString, "&")
	}

	//hash, MD5,SHA1,SHA256,SHA512
	//must create here for thread-safe
	hash := s.hashFunc()
	if s.hashKey != "" {
		// hmac(New, nil) is diffrent from New.
		hash = hmac.New(s.hashFunc, []byte(s.hashKey))
	}
	io.WriteString(hash, kvString)
	signStr := hex.EncodeToString(hash.Sum(nil))
	signStr = strings.ToUpper(signStr)

	return signStr == signature
}

//skip returns skip the key or not.
func (s *Signer) skip(key string) bool {
	for _, k := range s.omitKeys {
		if key == k {
			return true
		}
	}
	return false
}

// byName sorts field by name, breaking ties with depth,
// then breaking ties with "name came from toml tag", then
// breaking ties with index sequence.
type byName []*reflectx.FieldInfo

func (x byName) Len() int      { return len(x) }
func (x byName) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x byName) Less(i, j int) bool {
	return x[i].Name < x[j].Name
}
