package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func init() {
	registeredAuthers["gmail-regex"] = reflect.TypeOf(GoogleAuth{})
}

type GoogleAuth struct {
	config     *oauth2.Config
	emailRegex *regexp.Regexp
}

func (ga *GoogleAuth) UnmarshalJSON(data []byte) error {
	v := &struct {
		Credentials json.RawMessage
		EmailRegex  string
	}{}
	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}
	ga.config, err = google.ConfigFromJSON(v.Credentials, "openid", "email")
	if err != nil {
		return err
	}
	ga.emailRegex, err = regexp.Compile(v.EmailRegex)
	if err != nil {
		return err
	}
	return nil
}

func (ga GoogleAuth) OAuthConfig() *oauth2.Config {
	return ga.config
}

type googleIDToken struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func (ga GoogleAuth) Auth(code string) (string, error) {
	oauthConfig := ga.OAuthConfig()

	token, err := oauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return "", err
	}

	// Skipping signature verification since we just received this token directly from Google
	idToken, err := decodeJWS(token.Extra("id_token").(string))
	if err != nil {
		return "", err
	}

	if idToken.EmailVerified && ga.emailRegex.MatchString(idToken.Email) {
		return idToken.Email, nil
	}

	return "", fmt.Errorf("unauthorized")
}

// decodeJWS decodes a claim set from a JWS payload.  Does not verify the signature.
func decodeJWS(payload string) (*googleIDToken, error) {
	// decode returned id token to get expiry
	s := strings.Split(payload, ".")
	if len(s) < 2 {
		return nil, fmt.Errorf("jws: invalid token received")
	}
	decoded, err := base64Decode(s[1])
	if err != nil {
		return nil, err
	}
	token := &googleIDToken{}
	err = json.Unmarshal(decoded, token)
	return token, err
}

// base64Decode decodes the Base64url encoded string
func base64Decode(s string) ([]byte, error) {
	// add back missing padding
	switch len(s) % 4 {
	case 1:
		s += "==="
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
