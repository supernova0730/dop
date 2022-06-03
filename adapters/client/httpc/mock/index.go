package httpclient

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/rendau/dop/adapters/client/httpc"
	"github.com/rendau/dop/adapters/logger"
	"github.com/rendau/dop/dopErrs"
)

const (
	ErrPageNotFound = dopErrs.Err("page_not_found")
)

type St struct {
	lg logger.Lite

	requests  []*RequestSt
	responses map[string]ResponseSt
	mu        sync.Mutex
}

type RequestSt struct {
	Opts httpc.OptionsSt
	Raw  []byte
}

type ResponseSt struct {
	Obj interface{}
	Raw []byte
}

func New(lg logger.Lite) *St {
	return &St{
		lg: lg,

		requests:  []*RequestSt{},
		responses: map[string]ResponseSt{},
	}
}

func (c *St) SetResponses(responses map[string]ResponseSt) {
	c.mu.Lock()
	c.responses = map[string]ResponseSt{}
	c.mu.Unlock()

	for k, v := range responses {
		c.SetResponse(k, v)
	}
}

func (c *St) SetResponse(path string, response ResponseSt) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(response.Raw) == 0 && response.Obj != nil {
		var err error

		response.Raw, err = json.Marshal(response.Obj)
		if err != nil {
			c.lg.Errorw("Fail to marshal json", err)
		}
	}

	c.responses[path] = response
}

func (c *St) GetOptions() httpc.OptionsSt {
	return httpc.OptionsSt{}
}

func (c *St) Send(reqBody []byte, opts httpc.OptionsSt) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	request := &RequestSt{
		Opts: opts,
		Raw:  reqBody,
	}

	c.requests = append(c.requests, request)

	response, ok := c.responses[opts.Path]
	if !ok {
		c.lg.Infow("Httpc-mock, path not found", "path", opts.Path)
		return nil, ErrPageNotFound
	}

	return response.Raw, nil
}

func (c *St) SendJson(reqObj interface{}, opts httpc.OptionsSt) ([]byte, error) {
	if opts.Headers == nil {
		opts.Headers = http.Header{}
	}

	opts.Headers["Content-Type"] = []string{"application/json"}

	reqBody, err := json.Marshal(reqObj)
	if err != nil {
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
		return nil, err
	}

	return c.SendRecvJson(reqBody, repObj, opts)
}

func (c *St) GetRequests() []*RequestSt {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make([]*RequestSt, len(c.requests))

	for i, req := range c.requests {
		result[i] = req
	}

	return result
}

func (c *St) GetRequest(path string, obj interface{}) (*RequestSt, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, req := range c.requests {
		if req.Opts.Path != path {
			continue
		}

		if len(req.Raw) > 0 && obj != nil {
			err := json.Unmarshal(req.Raw, obj)
			if err != nil {
				c.lg.Errorw("Fail to unmarshal json", err)
				return nil, false
			}
		}

		return req, true
	}

	return nil, false
}

func (c *St) Clean() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requests = []*RequestSt{}
	c.responses = map[string]ResponseSt{}
}
