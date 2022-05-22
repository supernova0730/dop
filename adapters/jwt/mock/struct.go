package mock

type jwtCreateReqSt struct {
	Sub        string                 `json:"sub"`
	ExpSeconds int64                  `json:"exp_seconds"`
	Payload    map[string]interface{} `json:"payload"`
}

type jwtCreateRepSt struct {
	Token string `json:"token"`
}
