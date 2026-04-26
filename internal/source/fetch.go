// Copyright (c) 2026 godaily (Ainsley Clark)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package source

import (
	"context"
	"io"
	"net/http"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleydev/webkit/pkg/util/httputil"
	"github.com/pkg/errors"
)

// transformer is implemented by all per-source response item types.
type transformer interface {
	transform() news.Item
}

// httpClient is the shared HTTP client used by fetch. It can be replaced in
// tests to inject custom transports.
var httpClient = &http.Client{}

// fetch performs a GET request to url, checks for a 2xx status, reads the body
// into bytes, then calls unmarshal to decode it into T.
//
// Callers pass json.Unmarshal or xml.Unmarshal — both match the required
// func([]byte, any) error signature. Optional headers are merged onto the
// request; existing callers that pass none continue to work unchanged.
func fetch[T any](
	ctx context.Context,
	url string,
	name string,
	unmarshal func([]byte, any) error,
	headers ...http.Header,
) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return zero, errors.Wrap(err, name+" request creation failed")
	}

	for _, h := range headers {
		for k, vs := range h {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return zero, errors.Wrap(err, "fetch "+name)
	}
	defer resp.Body.Close()

	if !httputil.Is2xx(resp.StatusCode) {
		return zero, errors.Errorf("unexpected status code from %s: %d", name, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, errors.Wrap(err, "reading response")
	}

	var out T
	if err = unmarshal(data, &out); err != nil {
		return zero, errors.Wrap(err, "parsing response")
	}
	return out, nil
}

// transformAll maps a slice of items that implement transformer to []news.Item.
func transformAll[T transformer](items []T) []news.Item {
	out := make([]news.Item, len(items))
	for i, item := range items {
		out[i] = item.transform()
	}
	return out
}
