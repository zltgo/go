// client is used internally for testing. See readme for alternatives
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
)

// Client for graphql requests
type Client struct {
	url    string
	client *http.Client
}

// New creates a graphql client
func New(url string, client ...*http.Client) *Client {
	p := &Client{
		url: url,
	}

	if len(client) > 0 {
		p.client = client[0]
	} else {
		p.client = http.DefaultClient
	}
	return p
}

type Request struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
	Header        http.Header            `json:"-"`
	RemoteAddr    string				 `json:"-"`
}

type Option func(r *Request)

func Var(name string, value interface{}) Option {
	return func(r *Request) {
		if r.Variables == nil {
			r.Variables = map[string]interface{}{}
		}

		r.Variables[name] = value
	}
}

func Operation(name string) Option {
	return func(r *Request) {
		r.OperationName = name
	}
}

func (p *Client) MustPost(query string, response interface{}, options ...Option) {
	if err := p.Post(query, response, options...); err != nil {
		panic(err)
	}
}

func (p *Client) mkRequest(query string, options ...Option) Request {
	r := Request{
		Query:  query,
		Header: http.Header{},
	}

	for _, option := range options {
		option(&r)
	}

	return r
}

type ResponseData struct {
	Data       interface{}
	Errors     json.RawMessage
	Extensions map[string]interface{}
}

func (p *Client) Post(query string, response interface{}, options ...Option) (resperr error) {
	respDataRaw, resperr := p.RawPost(query, options...)
	if resperr != nil {
		return resperr
	}

	// we want to unpack even if there is an error, so we can see partial responses
	unpackErr := unpack(respDataRaw.Data, response)

	if respDataRaw.Errors != nil {
		return RawJsonError{respDataRaw.Errors}
	}
	return unpackErr
}

func (p *Client) RawPost(query string, options ...Option) (*ResponseData, error) {
	r := p.mkRequest(query, options...)
	requestBody, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("encode: %s", err.Error())
	}

	req, err := http.NewRequest("POST", p.url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("new request: %s", err.Error())
	}
	req.Header = r.Header
	req.RemoteAddr = r.RemoteAddr
	req.Header.Set("Content-Type", "application/json")
	rawResponse, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post: %s", err.Error())
	}
	defer func() {
		_ = rawResponse.Body.Close()
	}()

	if rawResponse.StatusCode >= http.StatusBadRequest {
		responseBody, _ := ioutil.ReadAll(rawResponse.Body)
		return nil, fmt.Errorf("http %d: %s", rawResponse.StatusCode, responseBody)
	}

	responseBody, err := ioutil.ReadAll(rawResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("read: %s", err.Error())
	}

	// decode it into map string first, let mapstructure do the final decode
	// because it can be much stricter about unknown fields.
	respDataRaw := &ResponseData{}
	err = json.Unmarshal(responseBody, &respDataRaw)
	if err != nil {
		return nil, fmt.Errorf("decode: %s", err.Error())
	}

	return respDataRaw, nil
}

type RawJsonError struct {
	json.RawMessage
}

func (r RawJsonError) Error() string {
	return string(r.RawMessage)
}

func unpack(data interface{}, into interface{}) error {
	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      into,
		TagName:     "json",
		ErrorUnused: true,
		ZeroFields:  true,
		DecodeHook: func(from reflect.Type, to reflect.Type, v interface{}) (interface{}, error) {
			if to == reflect.TypeOf(time.Time{}) {
				switch v := v.(type) {
				case float64:
					return time.Unix(int64(v), 0), nil
				case int64:
					return time.Unix(v, 0), nil
				}
			}

			if from == reflect.TypeOf(time.Time{}) && to.Kind() == reflect.Int64 {
				return v.(time.Time).Unix(), nil
			}

			return v, nil
		},
	})
	if err != nil {
		return fmt.Errorf("mapstructure: %s", err.Error())
	}

	return d.Decode(data)
}

//useless, does not work
func RandRemoteAddr(r *Request) {
	r.RemoteAddr = fmt.Sprintf("%d.%d.%d.%d:%d", rand.Intn(256),rand.Intn(256),rand.Intn(256),rand.Intn(256), rand.Intn(65535))
}