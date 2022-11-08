package jwts

import (
	"github.com/supernova0730/dop/adapters/client/httpc"
)

type St struct {
	httpc httpc.HttpC
}

func New(httpc httpc.HttpC) *St {
	return &St{
		httpc: httpc,
	}
}

func (p *St) Create(sub string, expSeconds int64, payload map[string]any) (string, error) {
	data := map[string]any{}

	for k, v := range payload {
		data[k] = v
	}

	if sub != "" {
		data["sub"] = sub
	}

	if expSeconds != 0 {
		data["exp_seconds"] = expSeconds
	}

	repObj := jwtCreateRepSt{}

	_, _, err := p.httpc.SendJsonRecvJson(data, &repObj, nil, httpc.OptionsSt{
		Method: "POST",
		Path:   "jwt",
	})
	if err != nil {
		return "", err
	}

	return repObj.Token, nil
}
