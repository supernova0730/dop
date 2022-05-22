package interfaces

type Sms interface {
	Send(phone string, msg string) bool
}
