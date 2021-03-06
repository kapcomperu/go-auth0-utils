package auth

import (
	"encoding/json"
	"errors"
	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"strings"
)

type Jwks struct {
	Keys []JSONWebKeys `json:"keys"`
}

type JSONWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

type CustomClaims struct {
	Scope       string   `json:"scope"`
	Email       string   `json:"https://kapcomperu.com/email"`
	Name        string   `json:"https://kapcomperu.com/name"`
	Roles       []string `json:"https://kapcomperu.com/roles"`
	Permissions []string `json:"permissions"`
	Subject     string   `json:"sub"`
	jwt.StandardClaims
}

type UserData struct {
	Name        string
	Email       string
	Subject     string
	Roles       []string
	Permissions []string
	Valid bool
}

type Response struct {
	Message string `json:"message"`
}

func CheckScope(headerAuth string,scope string, domain string) UserData {
	authHeaderParts := strings.Split(headerAuth, " ")
	tokenString := authHeaderParts[1]
	token, _ := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		cert, err := GetPemCert(token, domain)
		if err != nil {
			return nil, err
		}
		result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
		return result, nil
	})

	claims, ok := token.Claims.(*CustomClaims)
	userData := UserData{Name: claims.Name,Email: claims.Email,Subject: claims.Subject,Roles: claims.Roles,Permissions: claims.Permissions}
	hasScope := false
	if ok {
		result := userData.Permissions
		for i := range result {
			if result[i] == scope {
				hasScope = true
				break
			}
		}
	}
	userData.Valid = hasScope
	return userData
}

func GetPemCert(token *jwt.Token, domain string) (string, error) {
	cert := ""
	//resp, err := http.Get("https://" + os.Getenv("AUTH0_DOMAIN") + "/.well-known/jwks.json")
	resp, err := http.Get("https://" + domain + "/.well-known/jwks.json")

	if err != nil {
		return cert, err
	}
	defer resp.Body.Close()

	var jwks = Jwks{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		return cert, err
	}

	for k, _ := range jwks.Keys {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == "" {
		err := errors.New("Unable to find appropriate key.")
		return cert, err
	}

	return cert, nil
}

func CreateNewJwtmiddleware(audience string, domain string) *jwtmiddleware.JWTMiddleware {
	return jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			// Verify 'aud' claim
			aud := audience
			checkAud := token.Claims.(jwt.MapClaims).VerifyAudience(aud, false)
			if !checkAud {
				return token, errors.New("Invalid audience.")
			}
			// Verify 'iss' claim
			iss := "https://" + domain + "/"
			checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(iss, false)
			if !checkIss {
				return token, errors.New("Invalid issuer.")
			}

			cert, err := GetPemCert(token, domain)
			if err != nil {
				panic(err.Error())
			}

			result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
			return result, nil
		},
		SigningMethod: jwt.SigningMethodRS256,
	})
}
