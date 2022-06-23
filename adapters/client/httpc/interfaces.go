package httpc

type HttpC interface {
	GetOptions() OptionsSt
	Send(reqBody []byte, opts OptionsSt) ([]byte, error)
	SendJson(reqObj any, opts OptionsSt) ([]byte, error)
	SendRecvJson(reqBody []byte, repObj any, opts OptionsSt) ([]byte, error)
	SendJsonRecvJson(reqObj, repObj any, opts OptionsSt) ([]byte, error)
}
