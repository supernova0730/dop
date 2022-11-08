package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/supernova0730/dop/adapters/client/httpc"
	"github.com/supernova0730/dop/adapters/logger"
	"github.com/supernova0730/dop/dopErrs"
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

func (c *St) Send(reqBody []byte, opts httpc.OptionsSt) ([]byte, int, error) {
	opts = c.opts.GetMergedWith(opts)

	origLogFlags := opts.LogFlags

	var err error
	var repBody []byte
	var statusCode int

	for i := opts.RetryCount; i >= 0; i-- {
		if i == 0 {
			opts.LogFlags = origLogFlags
		} else {
			opts.LogFlags = origLogFlags | httpc.NoLogError
		}

		repBody, statusCode, err = c.send(reqBody, opts)
		if err != nil {
			if opts.RetryInterval > 0 {
				time.Sleep(opts.RetryInterval)
			}
			continue
		}

		return repBody, statusCode, nil
	}

	return repBody, statusCode, err
}

func (c *St) send(reqBody []byte, opts httpc.OptionsSt) ([]byte, int, error) {
	var err error

	uri := opts.BaseUrl + opts.Path

	logError := opts.LogFlags&httpc.NoLogError <= 0

	var req *http.Request

	if opts.Timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
		defer cancel()
		req, err = http.NewRequestWithContext(ctx, opts.Method, uri, bytes.NewBuffer(reqBody))
	} else {
		req, err = http.NewRequest(opts.Method, uri, bytes.NewBuffer(reqBody))
	}
	if err != nil {
		if logError {
			c.lg.Errorw(opts.BaseLogPrefix+opts.LogPrefix+"Fail to create http-request", err)
		}
		return nil, 0, err
	}

	// Headers
	if len(opts.BaseHeaders) > 0 || len(opts.Headers) > 0 {
		if len(opts.BaseHeaders) > 0 {
			for k, v := range opts.BaseHeaders {
				req.Header[k] = v
			}
		}
		if len(opts.Headers) > 0 {
			for k, v := range opts.Headers {
				req.Header[k] = v
			}
		}
	}

	// c.lg.Infow("dop request header", req.Header)

	var queryParamsString string

	// Query params
	if len(opts.BaseParams) > 0 || len(opts.Params) > 0 {
		qPars := url.Values{}
		if len(opts.BaseParams) > 0 {
			for k, v := range opts.BaseParams {
				qPars[k] = v
			}
		}
		if len(opts.Params) > 0 {
			for k, v := range opts.Params {
				qPars[k] = v
			}
		}
		queryParamsString = qPars.Encode()
		req.URL.RawQuery = queryParamsString
	}

	// Basic auth
	if opts.BasicAuthCreds != nil {
		req.SetBasicAuth(opts.BasicAuthCreds.Username, opts.BasicAuthCreds.Password)
	}

	if opts.LogFlags&httpc.LogRequest > 0 {
		c.lg.Infow(opts.BaseLogPrefix+opts.LogPrefix+"request: /"+opts.Path,
			"uri", uri,
			"params", queryParamsString,
			"body", string(reqBody),
		)
	}

	// Do request
	rep, err := opts.Client.Do(req)
	if err != nil {
		if logError {
			c.lg.Errorw(
				opts.BaseLogPrefix+opts.LogPrefix+"Fail to send http-request", err,
				"uri", uri,
				"params", queryParamsString,
				"req_body", string(reqBody),
			)
		}
		return nil, 0, err
	}
	defer rep.Body.Close()

	statusCode := rep.StatusCode

	// read response body
	repBody, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		if logError {
			c.lg.Errorw(
				opts.BaseLogPrefix+opts.LogPrefix+"Fail to read body", err,
				"uri", uri,
				"params", queryParamsString,
				"req_body", string(reqBody),
			)
		}
		return nil, statusCode, err
	}

	if rep.StatusCode < 200 || rep.StatusCode > 299 {
		if rep.StatusCode == 401 {
			if logError && opts.LogFlags&httpc.NoLogNotAuthorized <= 0 {
				c.lg.Errorw(
					opts.BaseLogPrefix+opts.LogPrefix+"Bad status code", nil,
					"status_code", rep.StatusCode,
					"rep_body", string(repBody),
					"uri", uri,
					"params", queryParamsString,
					"req_body", string(reqBody),
				)
			}
			return repBody, statusCode, dopErrs.NotAuthorized
		}
		if rep.StatusCode == 403 {
			if logError && opts.LogFlags&httpc.NoLogPermissionDenied <= 0 {
				c.lg.Errorw(
					opts.BaseLogPrefix+opts.LogPrefix+"Bad status code", nil,
					"status_code", rep.StatusCode,
					"rep_body", string(repBody),
					"uri", uri,
					"params", queryParamsString,
					"req_body", string(reqBody),
				)
			}
			return repBody, statusCode, dopErrs.PermissionDenied
		}
		if logError && opts.LogFlags&httpc.NoLogBadStatus <= 0 {
			c.lg.Errorw(
				opts.BaseLogPrefix+opts.LogPrefix+"Bad status code", nil,
				"status_code", rep.StatusCode,
				"rep_body", string(repBody),
				"uri", uri,
				"params", queryParamsString,
				"req_body", string(reqBody),
			)
		}
		return repBody, statusCode, dopErrs.BadStatusCode
	}

	if opts.LogFlags&httpc.LogResponse > 0 {
		c.lg.Infow(opts.BaseLogPrefix+opts.LogPrefix+"response: /"+opts.Path,
			"uri", uri,
			"params", queryParamsString,
			"req_body", string(reqBody),
			"body", string(repBody),
		)
	}

	return repBody, statusCode, nil
}

func (c *St) SendJson(reqObj any, opts httpc.OptionsSt) ([]byte, int, error) {
	if opts.Headers == nil {
		opts.Headers = http.Header{}
	}

	if len(c.opts.BaseHeaders.Values("Content-Type")) == 0 && len(c.opts.Headers.Values("Content-Type")) == 0 &&
		len(opts.BaseHeaders.Values("Content-Type")) == 0 && len(opts.Headers.Values("Content-Type")) == 0 {
		opts.Headers["Content-Type"] = []string{"application/json"}
	}

	reqBody, err := json.Marshal(reqObj)
	if err != nil {
		if opts.LogFlags&httpc.NoLogError <= 0 {
			c.lg.Errorw(opts.LogPrefix+"Fail to marshal json", err)
		}
		return nil, 0, err
	}

	return c.Send(reqBody, opts)
}

func (c *St) SendRecvJson(reqBody []byte, repObj any, statusRepObj map[int]any, opts httpc.OptionsSt) ([]byte, int, error) {
	if opts.Headers == nil {
		opts.Headers = http.Header{}
	}

	if len(c.opts.BaseHeaders.Values("Accept")) == 0 && len(c.opts.Headers.Values("Accept")) == 0 &&
		len(opts.BaseHeaders.Values("Accept")) == 0 && len(opts.Headers.Values("Accept")) == 0 {
		opts.Headers["Accept"] = []string{"application/json"}
	}

	repBody, statusCode, err := c.Send(reqBody, opts)
	if err != nil {
		if err != dopErrs.BadStatusCode && err != dopErrs.NotAuthorized && err != dopErrs.PermissionDenied {
			return repBody, statusCode, err
		}
	}

	if len(repBody) > 0 {
		if err == nil {
			if repObj != nil {
				err = json.Unmarshal(repBody, repObj)
				if err != nil {
					if opts.LogFlags&httpc.NoLogError <= 0 {
						c.lg.Errorw(
							opts.LogPrefix+"Fail to unmarshal body", err,
							"opts", opts,
							"req_body", string(reqBody),
							"rep_body", string(repBody),
						)
					}
				}
			}
		} else if statusCode > 0 {
			if statusRepObj != nil {
				if rObj, ok := statusRepObj[statusCode]; ok {
					err = json.Unmarshal(repBody, rObj)
					if err != nil {
						if opts.LogFlags&httpc.NoLogError <= 0 {
							c.lg.Errorw(
								opts.LogPrefix+"Fail to unmarshal body", err,
								"opts", opts,
								"req_body", string(reqBody),
								"rep_body", string(repBody),
							)
						}
					}
				}
			}
		}
	}

	return repBody, statusCode, err
}

func (c *St) SendJsonRecvJson(reqObj, repObj any, statusRepObj map[int]any, opts httpc.OptionsSt) ([]byte, int, error) {
	if opts.Headers == nil {
		opts.Headers = http.Header{}
	}

	if len(c.opts.BaseHeaders.Values("Content-Type")) == 0 && len(c.opts.Headers.Values("Content-Type")) == 0 &&
		len(opts.BaseHeaders.Values("Content-Type")) == 0 && len(opts.Headers.Values("Content-Type")) == 0 {
		opts.Headers["Content-Type"] = []string{"application/json"}
	}

	reqBody, err := json.Marshal(reqObj)
	if err != nil {
		if opts.LogFlags&httpc.NoLogError <= 0 {
			c.lg.Errorw(opts.LogPrefix+"Fail to marshal json", err)
		}
		return nil, 0, err
	}

	return c.SendRecvJson(reqBody, repObj, statusRepObj, opts)
}
