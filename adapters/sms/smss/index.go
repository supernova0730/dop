package smss

import (
	"github.com/rendau/dop/adapters/client/httpc"
)

type St struct {
	httpc httpc.HttpC
}

func New(httpc httpc.HttpC) *St {
	return &St{
		httpc: httpc,
	}
}

func (s *St) Send(phone string, msg string) bool {
	_, err := s.httpc.SendJson(SendReqSt{
		To:   phone,
		Text: msg,
		Sync: true,
	}, httpc.OptionsSt{
		Method: "POST",
		Path:   "send",
	})
	if err != nil {
		return false
	}

	return true
}
