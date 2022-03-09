// Copyright 2021 The Casdoor Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package object

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/casdoor/casdoor/util"
	"xorm.io/core"
)

type Code struct {
	Message string `xorm:"varchar(100)" json:"message"`
	Code    string `xorm:"varchar(100)" json:"code"`
}

type Token struct {
	Owner       string `xorm:"varchar(100) notnull pk" json:"owner"`
	Name        string `xorm:"varchar(100) notnull pk" json:"name"`
	CreatedTime string `xorm:"varchar(100)" json:"createdTime"`

	Application  string `xorm:"varchar(100)" json:"application"`
	Organization string `xorm:"varchar(100)" json:"organization"`
	User         string `xorm:"varchar(100)" json:"user"`

	Code          string `xorm:"varchar(100)" json:"code"`
	AccessToken   string `xorm:"mediumtext" json:"accessToken"`
	RefreshToken  string `xorm:"mediumtext" json:"refreshToken"`
	ExpiresIn     int    `json:"expiresIn"`
	Scope         string `xorm:"varchar(100)" json:"scope"`
	TokenType     string `xorm:"varchar(100)" json:"tokenType"`
	CodeChallenge string `xorm:"varchar(100)" json:"codeChallenge"`
	CodeIsUsed    bool   `json:"codeIsUsed"`
	CodeExpireIn  int64  `json:"codeExpireIn"`
}

type TokenWrapper struct {
	AccessToken  string `json:"access_token"`
	IdToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type IntrospectionResponse struct {
	Active    bool     `json:"active"`
	Scope     string   `json:"scope,omitempty"`
	ClientId  string   `json:"client_id,omitempty"`
	Username  string   `json:"username,omitempty"`
	TokenType string   `json:"token_type,omitempty"`
	Exp       int64    `json:"exp,omitempty"`
	Iat       int64    `json:"iat,omitempty"`
	Nbf       int64    `json:"nbf,omitempty"`
	Sub       string   `json:"sub,omitempty"`
	Aud       []string `json:"aud,omitempty"`
	Iss       string   `json:"iss,omitempty"`
	Jti       string   `json:"jti,omitempty"`
}

func GetTokenCount(owner, field, value string) int {
	session := GetSession(owner, -1, -1, field, value, "", "")
	count, err := session.Count(&Token{})
	if err != nil {
		panic(err)
	}

	return int(count)
}

func GetTokens(owner string) []*Token {
	tokens := []*Token{}
	err := adapter.Engine.Desc("created_time").Find(&tokens, &Token{Owner: owner})
	if err != nil {
		panic(err)
	}

	return tokens
}

func GetPaginationTokens(owner string, offset, limit int, field, value, sortField, sortOrder string) []*Token {
	tokens := []*Token{}
	session := GetSession(owner, offset, limit, field, value, sortField, sortOrder)
	err := session.Find(&tokens)
	if err != nil {
		panic(err)
	}

	return tokens
}

func getToken(owner string, name string) *Token {
	if owner == "" || name == "" {
		return nil
	}

	token := Token{Owner: owner, Name: name}
	existed, err := adapter.Engine.Get(&token)
	if err != nil {
		panic(err)
	}

	if existed {
		return &token
	}

	return nil
}

func getTokenByCode(code string) *Token {
	token := Token{Code: code}
	existed, err := adapter.Engine.Get(&token)
	if err != nil {
		panic(err)
	}

	if existed {
		return &token
	}

	return nil
}

func updateUsedByCode(token *Token) bool {
	affected, err := adapter.Engine.Where("code=?", token.Code).Cols("code_is_used").Update(token)
	if err != nil {
		panic(err)
	}

	return affected != 0
}

func GetToken(id string) *Token {
	owner, name := util.GetOwnerAndNameFromId(id)
	return getToken(owner, name)
}

func UpdateToken(id string, token *Token) bool {
	owner, name := util.GetOwnerAndNameFromId(id)
	if getToken(owner, name) == nil {
		return false
	}

	affected, err := adapter.Engine.ID(core.PK{owner, name}).AllCols().Update(token)
	if err != nil {
		panic(err)
	}

	return affected != 0
}

func AddToken(token *Token) bool {
	affected, err := adapter.Engine.Insert(token)
	if err != nil {
		panic(err)
	}

	return affected != 0
}

func DeleteToken(token *Token) bool {
	affected, err := adapter.Engine.ID(core.PK{token.Owner, token.Name}).Delete(&Token{})
	if err != nil {
		panic(err)
	}

	return affected != 0
}

func DeleteTokenByAceessToken(accessToken string) (bool, *Application) {
	token := Token{AccessToken: accessToken}
	existed, err := adapter.Engine.Get(&token)
	if err != nil {
		panic(err)
	}

	if !existed {
		return false, nil
	}
	application := getApplication(token.Owner, token.Application)
	affected, err := adapter.Engine.Where("access_token=?", accessToken).Delete(&Token{})
	if err != nil {
		panic(err)
	}

	return affected != 0, application
}

func GetTokenByAccessToken(accessToken string) *Token {
	//Check if the accessToken is in the database
	token := Token{AccessToken: accessToken}
	existed, err := adapter.Engine.Get(&token)
	if err != nil || !existed {
		return nil
	}
	return &token
}

func GetTokenByTokenAndApplication(token string, application string) *Token {
	tokenResult := Token{}
	existed, err := adapter.Engine.Where("(refresh_token = ? or access_token = ? ) and application = ?", token, token, application).Get(&tokenResult)
	if err != nil || !existed {
		return nil
	}
	return &tokenResult
}

func CheckOAuthLogin(clientId string, responseType string, redirectUri string, scope string, state string) (string, *Application) {
	if responseType != "code" && responseType != "token" && responseType != "id_token" {
		return fmt.Sprintf("error: grant_type: %s is not supported in this application", responseType), nil
	}

	application := GetApplicationByClientId(clientId)
	if application == nil {
		return "Invalid client_id", nil
	}

	validUri := false
	for _, tmpUri := range application.RedirectUris {
		if strings.Contains(redirectUri, tmpUri) {
			validUri = true
			break
		}
	}
	if !validUri {
		return fmt.Sprintf("Redirect URI: \"%s\" doesn't exist in the allowed Redirect URI list", redirectUri), application
	}

	// Mask application for /api/get-app-login
	application.ClientSecret = ""
	return "", application
}

func GetOAuthCode(userId string, clientId string, responseType string, redirectUri string, scope string, state string, nonce string, challenge string, host string) *Code {
	user := GetUser(userId)
	if user == nil {
		return &Code{
			Message: fmt.Sprintf("The user: %s doesn't exist", userId),
			Code:    "",
		}
	}
	if user.IsForbidden {
		return &Code{
			Message: "error: the user is forbidden to sign in, please contact the administrator",
			Code:    "",
		}
	}

	msg, application := CheckOAuthLogin(clientId, responseType, redirectUri, scope, state)
	if msg != "" {
		return &Code{
			Message: msg,
			Code:    "",
		}
	}

	accessToken, refreshToken, err := generateJwtToken(application, user, nonce, scope, host)
	if err != nil {
		panic(err)
	}

	if challenge == "null" {
		challenge = ""
	}

	token := &Token{
		Owner:         application.Owner,
		Name:          util.GenerateId(),
		CreatedTime:   util.GetCurrentTime(),
		Application:   application.Name,
		Organization:  user.Owner,
		User:          user.Name,
		Code:          util.GenerateClientId(),
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		ExpiresIn:     application.ExpireInHours * 60,
		Scope:         scope,
		TokenType:     "Bearer",
		CodeChallenge: challenge,
		CodeIsUsed:    false,
		CodeExpireIn:  time.Now().Add(time.Minute * 5).Unix(),
	}
	AddToken(token)

	return &Code{
		Message: "",
		Code:    token.Code,
	}
}

func GetOAuthToken(grantType string, clientId string, clientSecret string, code string, verifier string, scope string, username string, password string, host string) *TokenWrapper {
	application := GetApplicationByClientId(clientId)
	if application == nil {
		return &TokenWrapper{
			AccessToken: "error: invalid client_id",
			TokenType:   "",
			ExpiresIn:   0,
			Scope:       "",
		}
	}

	//Check if grantType is allowed in the current application
	if !IsGrantTypeValid(grantType, application.GrantTypes) {
		return &TokenWrapper{
			AccessToken: fmt.Sprintf("error: grant_type: %s is not supported in this application", grantType),
			TokenType:   "",
			ExpiresIn:   0,
			Scope:       "",
		}
	}

	var token *Token
	var err error
	switch grantType {
	case "authorization_code": // Authorization Code Grant
		token, err = GetAuthorizationCodeToken(application, clientSecret, code, verifier)
	case "password": //	Resource Owner Password Credentials Grant
		token, err = GetPasswordToken(application, username, password, scope, host)
	case "client_credentials": // Client Credentials Grant
		token, err = GetClientCredentialsToken(application, clientSecret, scope, host)
	}

	if err != nil {
		return &TokenWrapper{
			AccessToken: err.Error(),
			TokenType:   "",
			ExpiresIn:   0,
			Scope:       "",
		}
	}

	token.CodeIsUsed = true
	updateUsedByCode(token)
	tokenWrapper := &TokenWrapper{
		AccessToken:  token.AccessToken,
		IdToken:      token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresIn:    token.ExpiresIn,
		Scope:        token.Scope,
	}

	return tokenWrapper
}

func RefreshToken(grantType string, refreshToken string, scope string, clientId string, clientSecret string, host string) *TokenWrapper {
	// check parameters
	if grantType != "refresh_token" {
		return &TokenWrapper{
			AccessToken: "error: grant_type should be \"refresh_token\"",
			TokenType:   "",
			ExpiresIn:   0,
			Scope:       "",
		}
	}
	application := GetApplicationByClientId(clientId)
	if application == nil {
		return &TokenWrapper{
			AccessToken: "error: invalid client_id",
			TokenType:   "",
			ExpiresIn:   0,
			Scope:       "",
		}
	}
	if clientSecret != "" && application.ClientSecret != clientSecret {
		return &TokenWrapper{
			AccessToken: "error: invalid client_secret",
			TokenType:   "",
			ExpiresIn:   0,
			Scope:       "",
		}
	}
	// check whether the refresh token is valid, and has not expired.
	token := Token{RefreshToken: refreshToken}
	existed, err := adapter.Engine.Get(&token)
	if err != nil || !existed {
		return &TokenWrapper{
			AccessToken: "error: invalid refresh_token",
			TokenType:   "",
			ExpiresIn:   0,
			Scope:       "",
		}
	}

	cert := getCertByApplication(application)
	_, err = ParseJwtToken(refreshToken, cert)
	if err != nil {
		return &TokenWrapper{
			AccessToken: fmt.Sprintf("error: %s", err.Error()),
			TokenType:   "",
			ExpiresIn:   0,
			Scope:       "",
		}
	}
	// generate a new token
	user := getUser(application.Organization, token.User)
	if user.IsForbidden {
		return &TokenWrapper{
			AccessToken: "error: the user is forbidden to sign in, please contact the administrator",
			TokenType:   "",
			ExpiresIn:   0,
			Scope:       "",
		}
	}
	newAccessToken, newRefreshToken, err := generateJwtToken(application, user, "", scope, host)
	if err != nil {
		panic(err)
	}

	newToken := &Token{
		Owner:        application.Owner,
		Name:         util.GenerateId(),
		CreatedTime:  util.GetCurrentTime(),
		Application:  application.Name,
		Organization: user.Owner,
		User:         user.Name,
		Code:         util.GenerateClientId(),
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    application.ExpireInHours * 60,
		Scope:        scope,
		TokenType:    "Bearer",
	}
	AddToken(newToken)

	tokenWrapper := &TokenWrapper{
		AccessToken:  token.AccessToken,
		IdToken:      token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		ExpiresIn:    token.ExpiresIn,
		Scope:        token.Scope,
	}

	return tokenWrapper
}

// PkceChallenge: base64-URL-encoded SHA256 hash of verifier, per rfc 7636
func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(sum[:])
	return challenge
}

// Check if grantType is allowed in the current application
// authorization_code is allowed by default
func IsGrantTypeValid(method string, grantTypes []string) bool {
	if method == "authorization_code" {
		return true
	}
	for _, m := range grantTypes {
		if m == method {
			return true
		}
	}
	return false
}

// Authorization code flow
func GetAuthorizationCodeToken(application *Application, clientSecret string, code string, verifier string) (*Token, error) {
	if code == "" {
		return nil, errors.New("error: authorization code should not be empty")
	}

	token := getTokenByCode(code)
	if token == nil {
		return nil, errors.New("error: invalid authorization code")
	}
	if token.CodeIsUsed {
		// anti replay attacks
		return nil, errors.New("error: authorization code has been used")
	}

	if token.CodeChallenge != "" && pkceChallenge(verifier) != token.CodeChallenge {
		return nil, errors.New("error: incorrect code_verifier")
	}

	if application.ClientSecret != clientSecret {
		// when using PKCE, the Client Secret can be empty,
		// but if it is provided, it must be accurate.
		if token.CodeChallenge == "" {
			return nil, errors.New("error: invalid client_secret")
		} else {
			if clientSecret != "" {
				return nil, errors.New("error: invalid client_secret")
			}
		}
	}

	if application.Name != token.Application {
		return nil, errors.New("error: the token is for wrong application (client_id)")
	}

	if time.Now().Unix() > token.CodeExpireIn {
		// code must be used within 5 minutes
		return nil, errors.New("error: authorization code has expired")
	}
	return token, nil
}

// Resource Owner Password Credentials flow
func GetPasswordToken(application *Application, username string, password string, scope string, host string) (*Token, error) {
	user := getUser(application.Organization, username)
	if user == nil {
		return nil, errors.New("error: the user does not exist")
	}
	if user.Password != password {
		return nil, errors.New("error: invalid username or password")
	}
	if user.IsForbidden {
		return nil, errors.New("error: the user is forbidden to sign in, please contact the administrator")
	}
	accessToken, refreshToken, err := generateJwtToken(application, user, "", scope, host)
	if err != nil {
		return nil, err
	}
	token := &Token{
		Owner:        application.Owner,
		Name:         util.GenerateId(),
		CreatedTime:  util.GetCurrentTime(),
		Application:  application.Name,
		Organization: user.Owner,
		User:         user.Name,
		Code:         util.GenerateClientId(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    application.ExpireInHours * 60,
		Scope:        scope,
		TokenType:    "Bearer",
		CodeIsUsed:   true,
	}
	AddToken(token)
	return token, nil
}

// Client Credentials flow
func GetClientCredentialsToken(application *Application, clientSecret string, scope string, host string) (*Token, error) {
	if application.ClientSecret != clientSecret {
		return nil, errors.New("error: invalid client_secret")
	}
	nullUser := &User{
		Owner: application.Owner,
		Id:    application.GetId(),
		Name:  fmt.Sprintf("app/%s", application.Name),
	}
	accessToken, _, err := generateJwtToken(application, nullUser, "", scope, host)
	if err != nil {
		return nil, err
	}
	token := &Token{
		Owner:        application.Owner,
		Name:         util.GenerateId(),
		CreatedTime:  util.GetCurrentTime(),
		Application:  application.Name,
		Organization: application.Organization,
		User:         nullUser.Name,
		Code:         util.GenerateClientId(),
		AccessToken:  accessToken,
		ExpiresIn:    application.ExpireInHours * 60,
		Scope:        scope,
		TokenType:    "Bearer",
		CodeIsUsed:   true,
	}
	AddToken(token)
	return token, nil
}

// Implicit flow
func GetTokenByUser(application *Application, user *User, scope string, host string) (*Token, error) {
	accessToken, refreshToken, err := generateJwtToken(application, user, "", scope, host)
	if err != nil {
		return nil, err
	}
	token := &Token{
		Owner:        application.Owner,
		Name:         util.GenerateId(),
		CreatedTime:  util.GetCurrentTime(),
		Application:  application.Name,
		Organization: user.Owner,
		User:         user.Name,
		Code:         util.GenerateClientId(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    application.ExpireInHours * 60,
		Scope:        scope,
		TokenType:    "Bearer",
		CodeIsUsed:   true,
	}
	AddToken(token)
	return token, nil
}
