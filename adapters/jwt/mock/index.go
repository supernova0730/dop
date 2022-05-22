package mock

import (
	"encoding/base64"
	"encoding/json"

	"github.com/rendau/dop/adapters/logger"
)

type St struct {
	lg      logger.WarnAndError
	testing bool
}

func New(lg logger.WarnAndError, testing bool) *St {
	return &St{
		lg:      lg,
		testing: testing,
	}
}

func (p *St) Create(sub string, expSeconds int64, payload map[string]interface{}) (string, error) {
	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		p.lg.Errorw("Fail to marshal data", err)
		return "", err
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadRaw)

	return "XXX." + payloadB64 + ".YYY", nil
}
