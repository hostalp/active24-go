/*
Copyright 2023 Richard Kosegi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package active24

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"k8s.io/klog/v2"
)

type Option func(c *client)

func ApiEndpoint(ep string) Option {
	return func(c *client) {
		c.h.apiEndpoint = ep
	}
}

func Timeout(to time.Duration) Option {
	return func(c *client) {
		c.h.c.Timeout = to
	}
}

type ApiError interface {
	Error() error
	Response() *http.Response
}

type Client interface {
	// Dns provides interface to interact with DNS records
	Dns() Dns
}

func New(apiKey string, apiSecret string, opts ...Option) Client {
	c := &client{
		h: helper{
			apiEndpoint: "https://rest.active24.cz",
			authKey:     apiKey,
			authSecret:  apiSecret,
			c: http.Client{
				Timeout: time.Second * 10,
			},
			l:        klog.NewKlogr(),
			maxPages: 100, // default max pages to prevent infinite loops
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type client struct {
	h helper
}

func (c *client) Dns() Dns {
	return &dns{
		h: c.h,
	}
}

type apiError struct {
	err  error
	resp *http.Response
}

func (a *apiError) Error() error {
	return a.err
}

func (a *apiError) Response() *http.Response {
	return a.resp
}

type helper struct {
	apiEndpoint string
	authKey     string
	authSecret  string
	c           http.Client
	l           klog.Logger
	maxPages    int
}

func (ch *helper) getSignature(message, key string) string {
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

func (ch *helper) do(reqMethod string, reqPath string, reqBody io.Reader) (*http.Response, error) {
	return ch.doWithParams(reqMethod, reqPath, nil, reqBody)
}

func (ch *helper) doWithParams(reqMethod string, reqPath string, reqParams url.Values, reqBody io.Reader) (*http.Response, error) {
	reqPath = path.Join("/", reqPath)
	reqTimestamp := time.Now()
	canonicalRequest := fmt.Sprintf("%s %s %d", reqMethod, reqPath, reqTimestamp.Unix())
	authSignature := ch.getSignature(canonicalRequest, ch.authSecret)

	r, err := http.NewRequest(reqMethod, fmt.Sprintf("%s%s", ch.apiEndpoint, reqPath), reqBody)
	if err != nil {
		return nil, err
	}
	r.URL.RawQuery = reqParams.Encode()
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Date", reqTimestamp.UTC().Format(time.RFC3339))
	r.SetBasicAuth(ch.authKey, authSignature)
	ch.l.V(4).Info("Calling API", "method", reqMethod, "URL", r.URL.String())

	// Log the request
	/* dumpReq, dumpErr := httputil.DumpRequestOut(r, true)
	if dumpErr != nil {
		return nil, dumpErr
	}
	ch.l.V(4).Info("doWithParams", "REQUEST", string(dumpReq)) */

	resp, err := ch.c.Do(r)

	// Log the response
	/* dumpResp, dumpErr := httputil.DumpResponse(resp, true)
	if dumpErr != nil {
		return nil, dumpErr
	}
	ch.l.V(4).Info("doWithParams", "RESPONSE", string(dumpResp)) */

	return resp, err
}

func apiErr(resp *http.Response, err error) ApiError {
	if err == nil {
		return nil
	}
	return &apiError{
		err:  err,
		resp: resp,
	}
}
