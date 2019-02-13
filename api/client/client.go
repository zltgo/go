package client

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	"github.com/zltgo/api/bind"
	"github.com/zltgo/reflectx"
)

type Client struct {
	*http.Client
}

var Default = Client{&http.Client{}}

func init() {
	Default.Jar, _ = cookiejar.New(nil)
}

//执行get 请求, 根据content_type来自动解析到struct
func (m Client) Get(urlStr string, vs url.Values, ptr interface{}) (int, error) {
	r, err := NewFormRequest("GET", urlStr, nil)
	if err != nil {
		return http.StatusBadRequest, err
	}
	return m.Exec(r, ptr)
}

//return the StatusCode and error
//if error is nil, the ptr will changed by decode json
//if int is zero, the request send error
func (m Client) Exec(r *http.Request, ptr interface{}) (int, error) {
	res, err := m.Do(r)
	if err != nil {
		return 0, err
	}

	defer res.Body.Close()

	if res.StatusCode > 299 {
		return res.StatusCode, errors.New(res.Status)
	}

	typ := bind.GetContentType(res.Header)
	switch typ {
	case bind.MIMEJSON:
		if err = json.NewDecoder(res.Body).Decode(ptr); err != nil {
			return res.StatusCode, err
		}
	case bind.MIMEXML, bind.MIMEXML2:
		if err = xml.NewDecoder(res.Body).Decode(ptr); err != nil {
			return res.StatusCode, err
		}
	default:
		return res.StatusCode, fmt.Errorf("unexpected content type:%s", typ)
	}

	return res.StatusCode, nil
}

//Create a http.Request by method, urlStr and input parameter.
//"Content-Type" will set to "application/x-www-form-urlencoded".
func NewFormRequest(method, urlStr string, vs url.Values) (r *http.Request, err error) {
	switch method {
	case "GET", "DELETE":
		if len(vs) > 0 {
			urlStr = urlStr + "?" + vs.Encode()
		}
		r, err = http.NewRequest(method, urlStr, nil)
	case "POST", "PUT":
		var body io.Reader
		if len(vs) > 0 {
			body = strings.NewReader(vs.Encode())
		}
		r, err = http.NewRequest(method, urlStr, body)
	default:
		return nil, errors.New("unsupported method: " + method)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r, nil
}

//Create a http.Request by method, urlStr and input parameter.
//"Content-Type" will set to "application/json; charset=utf-8".
func NewJsonRequest(method, urlStr string, obj interface{}) (r *http.Request, err error) {
	switch method {
	case "GET", "DELETE":
		if obj != nil {
			vs := url.Values{}
			reflectx.StructToForm(obj, vs)
			if len(vs) > 0 {
				urlStr = urlStr + "?" + vs.Encode()
			}
		}
		r, err = http.NewRequest(method, urlStr, nil)
	case "POST", "PUT":
		b, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		r, err = http.NewRequest(method, urlStr, bytes.NewReader(b))
	default:
		return nil, errors.New("unsupported method: " + method)
	}
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	return r, nil
}

//Create a http.Request by method, urlStr and input parameter.
//"Content-Type" will set to "application/xml; charset=utf-8".
func NewXmlRequest(method, urlStr string, obj interface{}) (r *http.Request, err error) {
	switch method {
	case "GET", "DELETE":
		if obj != nil {
			vs := url.Values{}
			reflectx.StructToForm(obj, vs)
			if len(vs) > 0 {
				urlStr = urlStr + "?" + vs.Encode()
			}
		}
		r, err = http.NewRequest(method, urlStr, nil)
	case "POST", "PUT":
		b, err := xml.Marshal(obj)
		if err != nil {
			return nil, err
		}
		r, err = http.NewRequest(method, urlStr, bytes.NewReader(b))
	default:
		return nil, errors.New("unsupported method: " + method)
	}
	r.Header.Set("Content-Type", "application/xml; charset=utf-8")
	return r, nil
}

// create a http.Request for upload file.
func NewUploadRequest(url, fieldName, fileName, filePath string) (*http.Request, error) {
	buff := new(bytes.Buffer)
	w := multipart.NewWriter(buff)

	fw, err := w.CreateFormFile(fieldName, fileName)
	if err != nil {
		return nil, err
	}

	fh, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(fw, fh)
	if err != nil {
		return nil, err
	}

	contentType := w.FormDataContentType()
	w.Close()
	r, _ := http.NewRequest("POST", url, buff)
	r.Header.Set("Content-Type", contentType)

	return r, nil
}
