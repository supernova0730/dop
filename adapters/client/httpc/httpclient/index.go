package httpclient

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/rendau/dop/adapters/client/httpc"
	"github.com/rendau/dop/adapters/logger"
	"github.com/rendau/dop/errs"
)

type St struct {
	lg   logger.Lite
	opts httpc.OptionsSt
}

func New(lg logger.Lite, opts httpc.OptionsSt) *St {
	if opts.BaseUrl != "" {
		opts.BaseUrl = strings.TrimRight(opts.BaseUrl, "/") + "/"
	}

	return &St{
		lg:   lg,
		opts: opts,
	}
}

func (c *St) GetOptions() httpc.OptionsSt {
	return c.opts
}

func (c *St) Send(reqBody []byte, opts httpc.OptionsSt) ([]byte, error) {
	opts = c.opts.GetMergedWith(opts)

	uri := opts.BaseUrl + opts.Path

	if opts.LogFlags&httpc.LogRequest > 0 {
		c.lg.Infow(opts.BaseLogPrefix+opts.LogPrefix+" request: /"+opts.Path,
			"uri", uri,
			"body", string(reqBody),
		)
	}

	req, err := http.NewRequest(opts.Method, uri, bytes.NewBuffer(reqBody))
	if err != nil {
		c.lg.Errorw("Fail to create http-request", err)
		return nil, err
	}

	// Headers
	if len(opts.BaseHeaders) > 0 || len(opts.Headers) > 0 {
		if len(opts.BaseHeaders) > 0 {
			for k, v := range opts.BaseHeaders {
				req.Header.Set(k, v)
			}
		}
		if len(opts.Headers) > 0 {
			for k, v := range opts.Headers {
				req.Header.Set(k, v)
			}
		}
	}

	var queryParamsString string

	// Query params
	if len(opts.BaseParams) > 0 || len(opts.Params) > 0 {
		qPars := url.Values{}
		if len(opts.BaseParams) > 0 {
			for k, v := range opts.BaseParams {
				qPars.Set(k, v)
			}
		}
		if len(opts.Params) > 0 {
			for k, v := range opts.Params {
				qPars.Set(k, v)
			}
		}
		queryParamsString = qPars.Encode()
		req.URL.RawQuery = queryParamsString
	}

	// Basic auth
	if opts.BasicAuthCreds != nil {
		req.SetBasicAuth(opts.BasicAuthCreds.Username, opts.BasicAuthCreds.Password)
	}

	// Do request
	rep, err := opts.Client.Do(req)
	if err != nil {
		c.lg.Errorw(
			"Fail to send http-request", err,
			"uri", uri,
			"params", queryParamsString,
			"req_body", string(reqBody),
		)
		return nil, err
	}
	defer rep.Body.Close()

	// read response body
	repBody, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		c.lg.Errorw(
			"Fail to read body", err,
			"uri", uri,
			"params", queryParamsString,
			"req_body", string(reqBody),
		)
		return nil, err
	}

	if rep.StatusCode < 200 || rep.StatusCode > 299 {
		if rep.StatusCode == 401 || rep.StatusCode == 403 {
			return nil, errs.NotAuthorized
		}
		c.lg.Errorw(
			"Bad status code", nil,
			"status_code", rep.StatusCode,
			"rep_body", string(repBody),
			"uri", uri,
			"req_body", string(reqBody),
		)
		return nil, errs.BadStatusCode
	}

	if len(repBody) > 0 {
		if opts.LogFlags&httpc.LogResponse > 0 {
			c.lg.Infow(opts.BaseLogPrefix+opts.LogPrefix+" response: /"+opts.Path,
				"uri", uri,
				"body", string(repBody),
			)
		}
	}

	return repBody, nil
}

func (c *St) SendJson(reqObj interface{}, opts httpc.OptionsSt) ([]byte, error) {
	if opts.Headers == nil {
		opts.Headers = map[string]string{}
	}

	opts.Headers["Content-Type"] = "application/json"

	reqBody, err := json.Marshal(reqObj)
	if err != nil {
		c.lg.Errorw("Fail to marshal json", err)
		return nil, err
	}

	repBody, err := c.Send(reqBody, opts)
	if err != nil {
		return nil, err
	}

	return repBody, nil
}

func (c *St) SendRecvJson(reqBody []byte, repObj interface{}, opts httpc.OptionsSt) ([]byte, error) {
	if opts.Headers == nil {
		opts.Headers = map[string]string{}
	}

	opts.Headers["Accept"] = "application/json"

	repBody, err := c.Send(reqBody, opts)
	if err != nil {
		return nil, err
	}

	if len(repBody) > 0 {
		if repObj != nil {
			err = json.Unmarshal(repBody, repObj)
			if err != nil {
				c.lg.Errorw(
					"Fail to unmarshal body", err,
					"opts", opts,
					"req_body", string(reqBody),
				)
				return nil, err
			}
		}
	}

	return repBody, nil
}

func (c *St) SendJsonRecvJson(reqObj, repObj interface{}, opts httpc.OptionsSt) ([]byte, error) {
	if opts.Headers == nil {
		opts.Headers = map[string]string{}
	}

	opts.Headers["Content-Type"] = "application/json"

	reqBody, err := json.Marshal(reqObj)
	if err != nil {
		c.lg.Errorw("Fail to marshal json", err)
		return nil, err
	}

	return c.SendRecvJson(reqBody, repObj, opts)
}
