package httpclient

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rendau/dop/adapters/client/httpc"
	"github.com/rendau/dop/adapters/logger"
	"github.com/rendau/dop/dopErrs"
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

	origLogFlags := opts.LogFlags

	for i := opts.RetryCount; i >= 0; i-- {
		if i == 0 {
			opts.LogFlags = origLogFlags
		} else {
			opts.LogFlags = origLogFlags | httpc.NoLogError
		}

		repBody, err := c.send(reqBody, opts)
		if err != nil {
			if opts.RetryInterval > 0 {
				time.Sleep(opts.RetryInterval)
			}
			continue
		}

		return repBody, nil
	}

	return nil, nil
}

func (c *St) send(reqBody []byte, opts httpc.OptionsSt) ([]byte, error) {
	uri := opts.BaseUrl + opts.Path

	logError := opts.LogFlags&httpc.NoLogError <= 0

	if opts.LogFlags&httpc.LogRequest > 0 {
		c.lg.Infow(opts.BaseLogPrefix+opts.LogPrefix+"request: /"+opts.Path,
			"uri", uri,
			"body", string(reqBody),
		)
	}

	req, err := http.NewRequest(opts.Method, uri, bytes.NewBuffer(reqBody))
	if err != nil {
		if logError {
			c.lg.Errorw(opts.BaseLogPrefix+opts.LogPrefix+"Fail to create http-request", err)
		}
		return nil, err
	}

	// Headers
	if len(opts.BaseHeaders) > 0 || len(opts.Headers) > 0 {
		if len(opts.BaseHeaders) > 0 {
			req.Header = opts.BaseHeaders
		}
		if len(opts.Headers) > 0 {
			req.Header = opts.Headers
		}
	}

	var queryParamsString string

	// Query params
	if len(opts.BaseParams) > 0 || len(opts.Params) > 0 {
		qPars := url.Values{}
		if len(opts.BaseParams) > 0 {
			qPars = opts.BaseParams
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
		return nil, err
	}
	defer rep.Body.Close()

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
		return nil, err
	}

	if rep.StatusCode < 200 || rep.StatusCode > 299 {
		if rep.StatusCode == 401 || rep.StatusCode == 403 {
			return nil, dopErrs.NotAuthorized
		}
		if logError {
			c.lg.Errorw(
				opts.BaseLogPrefix+opts.LogPrefix+"Bad status code", nil,
				"status_code", rep.StatusCode,
				"rep_body", string(repBody),
				"uri", uri,
				"req_body", string(reqBody),
			)
		}
		return nil, dopErrs.BadStatusCode
	}

	if len(repBody) > 0 {
		if opts.LogFlags&httpc.LogResponse > 0 {
			c.lg.Infow(opts.BaseLogPrefix+opts.LogPrefix+"response: /"+opts.Path,
				"uri", uri,
				"body", string(repBody),
			)
		}
	}

	return repBody, nil
}

func (c *St) SendJson(reqObj interface{}, opts httpc.OptionsSt) ([]byte, error) {
	if opts.Headers == nil {
		opts.Headers = http.Header{}
	}

	opts.Headers["Content-Type"] = []string{"application/json"}

	reqBody, err := json.Marshal(reqObj)
	if err != nil {
		if opts.LogFlags&httpc.NoLogError <= 0 {
			c.lg.Errorw(opts.LogPrefix+"Fail to marshal json", err)
		}
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
		opts.Headers = http.Header{}
	}

	opts.Headers["Accept"] = []string{"application/json"}

	repBody, err := c.Send(reqBody, opts)
	if err != nil {
		return nil, err
	}

	if len(repBody) > 0 {
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
				return nil, err
			}
		}
	}

	return repBody, nil
}

func (c *St) SendJsonRecvJson(reqObj, repObj interface{}, opts httpc.OptionsSt) ([]byte, error) {
	if opts.Headers == nil {
		opts.Headers = http.Header{}
	}

	opts.Headers["Content-Type"] = []string{"application/json"}

	reqBody, err := json.Marshal(reqObj)
	if err != nil {
		if opts.LogFlags&httpc.NoLogError <= 0 {
			c.lg.Errorw(opts.LogPrefix+"Fail to marshal json", err)
		}
		return nil, err
	}

	return c.SendRecvJson(reqBody, repObj, opts)
}
