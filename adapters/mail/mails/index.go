package mails

import (
	"github.com/rendau/dop/adapters/client/httpc"
	"github.com/rendau/dop/adapters/mail"
)

type St struct {
	httpc httpc.HttpC
}

func New(httpc httpc.HttpC) *St {
	return &St{
		httpc: httpc,
	}
}

func (m *St) Send(data *mail.SendReqSt) bool {
	_, err := m.httpc.SendJson(data, httpc.OptionsSt{
		Method: "POST",
		Path:   "send",
	})
	if err != nil {
		return false
	}

	return true
}
