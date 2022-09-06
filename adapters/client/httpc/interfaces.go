package httpc

type HttpC interface {
	GetOptions() OptionsSt
	Send(reqBody []byte, opts OptionsSt) ([]byte, int, error)
	SendJson(reqObj any, opts OptionsSt) ([]byte, int, error)
	SendRecvJson(reqBody []byte, repObj any, statusRepObj map[int]any, opts OptionsSt) ([]byte, int, error)
	SendJsonRecvJson(reqObj, repObj any, statusRepObj map[int]any, opts OptionsSt) ([]byte, int, error)
}
