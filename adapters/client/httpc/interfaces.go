package httpc

type HttpC interface {
	GetOptions() OptionsSt
	Send(reqBody []byte, opts OptionsSt) ([]byte, error)
	SendJson(reqObj interface{}, opts OptionsSt) ([]byte, error)
	SendRecvJson(reqBody []byte, repObj interface{}, opts OptionsSt) ([]byte, error)
	SendJsonRecvJson(reqObj, repObj interface{}, opts OptionsSt) ([]byte, error)
}
