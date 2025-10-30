package auth

import (
	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"time"
	"log"
	"errors"
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
	/*
	type RegisteredClaims struct {
	// the `iss` (Issuer) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.1
	Issuer string `json:"iss,omitempty"`

	// the `sub` (Subject) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.2
	Subject string `json:"sub,omitempty"`
	// //temp
	// the `aud` (Audience) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.3
	//Audience ClaimStrings `json:"aud,omitempty"`

	// the `exp` (Expiration Time) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.4
	ExpiresAt *NumericDate `json:"exp,omitempty"`

	// the `nbf` (Not Before) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.5
	//NotBefore *NumericDate `json:"nbf,omitempty"`

	// the `iat` (Issued At) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.6
	IssuedAt *NumericDate `json:"iat,omitempty"`
	//
	// the `jti` (JWT ID) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.7
	ID string `json:"jti,omitempty"`
	}
	*/

	/*
	claims := MyCustomClaims{
		"bar",
		jwt.RegisteredClaims{
			// A usual scenario is to set the expiration time relative to the current time
			//ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),//UTC?
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "chirpy",
			Subject:   string(userID),
			ID:        "1",
			Audience:  []string{"somebody_else"},
		},
	}
	*/

	// Create the Claims
	claims := &jwt.RegisteredClaims{
		//ExpiresAt: jwt.NewNumericDate(time.Unix(1516239022, 0)),
		//Issuer:    "test",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),//UTC?
		Issuer:    "chirpy",
		Subject:   userID.String(),//[]string()
	}

	//t = jwt.New(jwt.SigningMethodHS256)
	//s = t.SignedString(key)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(tokenSecret)//mySigningKey
	if err != nil {
		return "error signing string", err
	}
	return ss, err
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	type NumericDate struct {
		time.Time
	}
	type Claims interface {
		GetExpirationTime() (*NumericDate, error)
		GetIssuedAt() (*NumericDate, error)
		//GetNotBefore() (*NumericDate, error)
		GetIssuer() (string, error)
		GetSubject() (string, error)
		//GetAudience() (ClaimStrings, error)
	}
	type SigningMethod interface {
		Verify(signingString string, sig []byte, key any) error // Returns nil if signature is valid
		Sign(signingString string, key any) ([]byte, error)     // Returns signature or error
		Alg() string                                            // returns the alg identifier for this method (example: 'HS256')
	}
	type Token struct {
		Raw       string         // Raw contains the raw token.  Populated when you [Parse] a token
		Method    SigningMethod  // Method is the signing method used or to be used
		Header    map[string]any // Header is the first segment of the token in decoded form
		Claims    Claims         // Claims is the second segment of the token in decoded form
		Signature []byte         // Signature is the third segment of the token in decoded form.  Populated when you Parse a token
		Valid     bool           // Valid specifies if the token is valid.  Populated when you Parse/Verify a token
	}

	type RegisteredClaims struct {//from above
	// the `iss` (Issuer) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.1
	Issuer string `json:"iss,omitempty"`

	// the `sub` (Subject) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.2
	Subject string `json:"sub,omitempty"`
	/* //temp
	// the `aud` (Audience) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.3
	Audience ClaimStrings `json:"aud,omitempty"`

	// the `exp` (Expiration Time) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.4
	ExpiresAt *NumericDate `json:"exp,omitempty"`

	// the `nbf` (Not Before) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.5
	NotBefore *NumericDate `json:"nbf,omitempty"`

	// the `iat` (Issued At) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.6
	IssuedAt *NumericDate `json:"iat,omitempty"`
	*/
	// the `jti` (JWT ID) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.7
	ID string `json:"jti,omitempty"`
	}

	type MyCustomClaims struct {
		Foo string `json:"foo"`
		jwt.RegisteredClaims
	}

	if tokenString == "" {
		return uuid.Nil, errors.New("empty token")
	}
	if tokenSecret == "" {
		return uuid.Nil, errors.New("empty token secret")
	}
	
	token, err := jwt.ParseWithClaims(tokenString, &MyCustomClaims{}, func(token *jwt.Token) (any, error) {
		return []byte("AllYourBase"), nil
	}, jwt.WithLeeway(5*time.Second))
	if err != nil {
		log.Fatal(err)
		return uuid.Nil, err
	} else if claims, ok := token.Claims.(*MyCustomClaims); ok {
		id := claims.Subject
		subjectuuid, err := uuid.Parse(id)
		if err != nil{
			return uuid.Nil, err
		}
		return subjectuuid, nil
	} else {
		log.Fatal("unknown claims type, cannot proceed")
		return uuid.Nil, err
	}
}
