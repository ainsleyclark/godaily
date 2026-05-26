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

package metrics

import (
	"net/http"

	"github.com/ainsleyclark/godaily/pkg/api"
	"github.com/ainsleyclark/godaily/pkg/domain/engagement"
	"github.com/ainsleydev/webkit/pkg/webkit"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type sourcesRequest struct {
	From   string `schema:"from"`
	To     string `schema:"to"`
	Period string `schema:"period"`
	Limit  int    `schema:"limit"`
}

func (req sourcesRequest) validate() error {
	return validation.ValidateStruct(
		&req,
		validation.Field(&req.Limit, validation.Min(0), validation.Max(api.MaxMetricsLimit)),
	)
}

// Sources handles GET /metrics/sources.
// Returns total clicks aggregated by item source.
func (h *Handler) Sources(c *webkit.Context) error {
	var req sourcesRequest
	if err := api.Decoder.Decode(&req, c.Request.URL.Query()); err != nil {
		return webkit.NewError(http.StatusBadRequest, "invalid query parameters")
	}
	if err := req.validate(); err != nil {
		return webkit.NewError(http.StatusBadRequest, err.Error())
	}

	from, to, err := api.ParseDateWindow(req.From, req.To, req.Period)
	if err != nil {
		return webkit.NewError(http.StatusBadRequest, err.Error())
	}

	limit := req.Limit
	if limit == 0 {
		limit = api.DefaultMetricsLimit
	}

	rows, err := h.metricsRepo.SourceList(c.Context(), engagement.MetricsFilter{From: from, To: to, Limit: limit})
	if err != nil {
		return webkit.NewError(http.StatusInternalServerError, "failed to fetch source metrics")
	}

	return c.JSON(http.StatusOK, map[string]any{"data": rows})
}
