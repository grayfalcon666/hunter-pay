package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const minSecretKeySize = 32

type JWTMaker struct {
	secretKey string
}

func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, errors.New("secret key too short")
	}
	return &JWTMaker{secretKey}, nil
}

func (maker *JWTMaker) CreateToken(username string, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(username, duration)
	if err != nil {
		return "", payload, err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	token, err := jwtToken.SignedString([]byte(maker.secretKey))
	return token, payload, err
}

func (maker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, errors.New("invalid token method")
		}
		return []byte(maker.secretKey), nil
	}

	var payload Payload
	//把 JWT 字符串解析后，将里面的 Claims 数据（比如 username、expiresAt）填充到payload 结构体中
	jwtToken, err := jwt.ParseWithClaims(token, &payload, keyFunc)
	if err != nil {
		return nil, err
	}

	if !jwtToken.Valid {
		return nil, errors.New("invalid token")
	}

	return &payload, nil
}
