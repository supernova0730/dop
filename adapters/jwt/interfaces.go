package jwt

type Jwt interface {
	Create(sub string, expSeconds int64, payload map[string]interface{}) (string, error)
}
