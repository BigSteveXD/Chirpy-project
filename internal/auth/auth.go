package auth

import (
	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"time"
	//"log"
	"errors"
	"net/http"
	"strings"
	"fmt"
	"crypto/rand"
	"encoding/hex"
)

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return match, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    string("chirpy-access"),//TokenTypeAccess
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		//ExpiresAt: jwt.NewNumericDate(time.Unix(1516239022, 0)),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	ss, err := token.SignedString([]byte(tokenSecret))//signingKey
	if err != nil {
		return "error signing string", err
	}
	return ss, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	/*
	if tokenString == "" {
		return uuid.Nil, errors.New("empty token string")
	}
	if tokenSecret == "" {
		return uuid.Nil, errors.New("empty token secret")
	}
	
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	},)//, jwt.WithLeeway(5*time.Second))
	if err != nil {
		log.Fatal(err)
		return uuid.Nil, err
	}
	
	userIDString, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return uuid.Nil, err
	}
	if issuer != string("chirpy-access") {//TokenTypeAccess
		return uuid.Nil, errors.New("invalid issuer")
	}

	id, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid userID: %w", err)
	}
	return id, nil
	*/
	claimsStruct := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claimsStruct,
		func(token *jwt.Token) (interface{}, error) { return []byte(tokenSecret), nil },
	)
	if err != nil {
		return uuid.Nil, err
	}

	userIDString, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return uuid.Nil, err
	}
	if issuer != string("chirpy-access") {
		return uuid.Nil, errors.New("invalid issuer")
	}

	id, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID: %w", err)
	}
	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	tokenStr := headers.Get("Authorization")
	if tokenStr == "" {
		return "recieved empty token string", fmt.Errorf("empty token string")
	}
	/*
	tokenStr = strings.Trim(tokenStr, "Bearer")
	tokenStr = strings.TrimSpace(tokenStr)
	return tokenStr, nil
	*/
	splitAuth := strings.Split(tokenStr, " ")//TEMP
	if len(splitAuth) < 2 || splitAuth[0] != "Bearer" {//TEMP
		return "", errors.New("malformed authorization header")//TEMP
	}//TEMP
	return splitAuth[1], nil//TEMP
}


func MakeRefreshToken() (string, error) {
	//rand.Read to generate 32 bytes (256 bits) of random data from the crypto/rand package (math/rand's Read function is deprecated).
	//hex.EncodeToString to convert the random data to a hex string
	randBytes := make([]byte, 32)
	_, err := rand.Read(randBytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(randBytes), nil
}
