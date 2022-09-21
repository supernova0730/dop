package jwt

type Jwk interface {
	Validate(token string) (bool, error)
}
