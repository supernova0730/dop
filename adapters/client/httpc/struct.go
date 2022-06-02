package httpc

import (
	"net/http"
	"net/url"
	"time"
)

type OptionsSt struct {
	Client         *http.Client
	BaseUrl        string
	BaseParams     url.Values
	BaseHeaders    http.Header
	BaseLogPrefix  string
	BasicAuthCreds *BasicAuthCredsSt

	Method        string
	Path          string
	Params        url.Values
	Headers       http.Header
	LogFlags      int
	LogPrefix     string
	RetryCount    int
	RetryInterval time.Duration
}

type BasicAuthCredsSt struct {
	Username string
	Password string
}

func (o OptionsSt) GetMergedWith(v OptionsSt) OptionsSt {
	res := OptionsSt{
		Client:         o.Client,
		BaseUrl:        o.BaseUrl,
		BaseParams:     o.BaseParams,
		BaseHeaders:    o.BaseHeaders,
		BaseLogPrefix:  o.BaseLogPrefix,
		BasicAuthCreds: o.BasicAuthCreds,
		Method:         o.Method,
		Path:           o.Path,
		Params:         o.Params,
		Headers:        o.Headers,
		LogFlags:       o.LogFlags,
		LogPrefix:      o.LogPrefix,
		RetryCount:     o.RetryCount,
	}

	if v.Client != nil {
		res.Client = v.Client
	}
	if v.BaseUrl != "" {
		if v.BaseUrl == "-" {
			res.BaseUrl = ""
		} else {
			res.BaseUrl = v.BaseUrl
		}
	}
	if v.BaseParams != nil {
		res.BaseParams = v.BaseParams
	}
	if v.BaseHeaders != nil {
		res.BaseHeaders = v.BaseHeaders
	}
	if v.BaseLogPrefix != "" {
		if v.BaseLogPrefix == "-" {
			res.BaseLogPrefix = ""
		} else {
			res.BaseLogPrefix = v.BaseLogPrefix
		}
	}
	if v.BasicAuthCreds != nil {
		res.BasicAuthCreds = v.BasicAuthCreds
	}
	if v.Method != "" {
		if v.Method == "-" {
			res.Method = ""
		} else {
			res.Method = v.Method
		}
	}
	if v.Path != "" {
		if v.Path == "-" {
			res.Path = ""
		} else {
			res.Path = v.Path
		}
	}
	if v.Params != nil {
		res.Params = v.Params
	}
	if v.Headers != nil {
		res.Headers = v.Headers
	}
	if v.LogFlags != 0 {
		if v.LogFlags < 0 {
			res.LogFlags = 0
		} else {
			res.LogFlags = v.LogFlags
		}
	}
	if v.LogPrefix != "" {
		if v.LogPrefix == "-" {
			res.LogPrefix = ""
		} else {
			res.LogPrefix = v.LogPrefix
		}
	}
	if v.RetryCount != 0 {
		if v.RetryCount < 0 {
			res.RetryCount = 0
		} else {
			res.RetryCount = v.RetryCount
		}
	}
	if v.RetryInterval != 0 {
		if v.RetryInterval < 0 {
			res.RetryInterval = 0
		} else {
			res.RetryInterval = v.RetryInterval
		}
	}

	return res
}
