package provision

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/txn2/ack"
	"github.com/txn2/es/v2"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const IdxUser = "user"
const EncCost = 12
const RedactMsg = "REDACTED"

// User defines a user object
type User struct {
	Id            string   `json:"id" json:"id" mapstructure:"id"`
	Description   string   `json:"description" yaml:"description" mapstructure:"description"`
	DisplayName   string   `json:"display_name" yaml:"displayName" mapstructure:"display_name"`
	Active        bool     `json:"active" yaml:"active" mapstructure:"active"`
	Sysop         bool     `json:"sysop" yaml:"sysop" mapstructure:"sysop"`
	Password      string   `json:"password" yaml:"password" mapstructure:"password"`
	Sections      []string `json:"sections" yaml:"sections" mapstructure:"sections"`
	SectionsAll   bool     `json:"sections_all" yaml:"sectionsAll" mapstructure:"sections_all"`
	Accounts      []string `json:"accounts" yaml:"accounts" mapstructure:"accounts"`
	AdminAccounts []string `json:"admin_accounts" yaml:"adminAccounts" mapstructure:"admin_accounts"`
}

// UserResult returned from Elastic
type UserResult struct {
	es.Result
	Source User `json:"_source"`
}

// UserTokenResultAck
type UserResultAck struct {
	ack.Ack
	Payload UserResult `json:"payload"`
}

// UserTokenResult
type UserTokenResult struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

// UserTokenResultAck
type UserTokenResultAck struct {
	ack.Ack
	Payload UserTokenResult `json:"payload"`
}

// Auth for authenticating users
type Auth struct {
	Id       string `json:"id"`
	Password string `json:"password"`
}

// UpsertUser inserts or updates a user record. Elasticsearch
// treats documents as immutable.
func (a *Api) UpsertUser(user *User) (int, es.Result, *es.ErrorResponse, error) {
	a.Logger.Info("Upsert user record", zap.String("id", user.Id), zap.String("display_name", user.DisplayName))

	// attempt to encrypt the password if one was provided
	// otherwise populate with existing
	err := user.CheckEncryptPassword(a)
	if err != nil {
		return 500, es.Result{}, nil, err
	}

	return a.Elastic.PutObj(fmt.Sprintf("%s/_doc/%s", a.IdxPrefix+IdxUser, user.Id), user)
}

// UpsertUserHandler
func (a *Api) UpsertUserHandler(c *gin.Context) {
	ak := ack.Gin(c)

	user := &User{}
	err := ak.UnmarshalPostAbort(user)
	if err != nil {
		a.Logger.Error("Upsert failure.", zap.Error(err))
		return
	}

	code, esResult, errorResponse, err := a.UpsertUser(user)
	if err != nil {
		a.Logger.Error("Upsert failure.", zap.Error(err))
		ak.SetPayloadType("ErrorMessage")
		ak.SetPayload("there was a problem upserting the user")
		if errorResponse != nil {
			a.Logger.Error("EsErrorResponse", zap.String("es_error_response", errorResponse.Message))
			ak.SetPayloadType("EsErrorResponse")
			ak.SetPayload(errorResponse)
		}
		ak.GinErrorAbort(500, "UpsertError", err.Error())
		return
	}

	if code < 200 || code >= 300 {

		a.Logger.Error("Es returned a non 200")
		ak.SetPayloadType("EsError")
		ak.SetPayload(esResult)
		ak.GinErrorAbort(500, "EsError", "Es returned a non 200")
		return
	}

	ak.SetPayloadType("EsResult")
	ak.GinSend(esResult)
}

// GetUser
func (a *Api) GetUser(id string) (int, *UserResult, error) {

	userResult := &UserResult{}

	code, ret, err := a.Elastic.Get(fmt.Sprintf("%s/_doc/%s", a.IdxPrefix+IdxUser, id))
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		return code, userResult, err
	}

	err = json.Unmarshal(ret, userResult)
	if err != nil {
		return code, userResult, err
	}

	return code, userResult, nil
}

// GetUserHandler gets a user by ID
func (a *Api) GetUserHandler(c *gin.Context) {
	ak := ack.Gin(c)

	id := c.Param("id")
	code, userResult, err := a.GetUser(id)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		ak.SetPayloadType("EsError")
		ak.SetPayload("Error communicating with database.")
		ak.GinErrorAbort(500, "EsError", err.Error())
		return
	}

	userResult.Source.Password = RedactMsg

	if code >= 400 && code < 500 {
		ak.SetPayload("User " + id + " not found.")
		ak.GinErrorAbort(404, "UserNotFound", "User not found")
		return
	}

	ak.SetPayloadType("UserResult")
	ak.GinSend(userResult)
}

// AuthUser authenticates a user with id and password
func (a *Api) AuthUser(auth Auth) (*UserResult, bool, error) {

	code, userResult, err := a.GetUser(auth.Id)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		return nil, false, err
	}

	if code >= 400 && code < 500 {
		a.Logger.Warn("User " + auth.Id + " not found")
		return nil, false, nil
	}

	if code >= 500 {
		a.Logger.Error("Received 500 code from database.")
		return nil, false, errors.New("received 500 code from database")
	}

	err = bcrypt.CompareHashAndPassword([]byte(userResult.Source.Password), []byte(auth.Password))
	if err != nil {
		return userResult, false, nil
	}

	return userResult, true, nil
}

// AuthUserHandler
func (a *Api) AuthUserHandler(c *gin.Context) {
	ak := ack.Gin(c)

	auth := &Auth{}
	err := ak.UnmarshalPostAbort(auth)
	if err != nil {
		a.Logger.Error("AuthUser failure.", zap.Error(err))
		return
	}

	foundUser, ok, err := a.AuthUser(*auth)
	if err != nil {
		a.Logger.Error("Auth error", zap.Error(err))
		ak.GinErrorAbort(500, "AuthError", err.Error())
		return
	}

	if foundUser == nil {
		ak.SetPayloadType("AuthFailResult")
		ak.GinErrorAbort(401, "AuthFailure", "User account not found.")
		return
	}

	foundUser.Source.Password = RedactMsg

	if ok {
		tkn, err := a.Config.Token.GetToken(foundUser.Source)
		if err != nil {
			a.Logger.Error("TokenFailResult", zap.Error(err))
			ak.SetPayloadType("TokenFailResult")
			ak.GinErrorAbort(401, "AuthFailure", "Filed to generate token.")
			return
		}

		if c.Query("raw") == "true" {
			c.Data(200, "text/plain", []byte(tkn))
			return
		}

		ak.SetPayloadType("UserTokenResult")
		ak.GinSend(UserTokenResult{
			User:  foundUser.Source,
			Token: tkn,
		})
		return
	}

	ak.GinErrorAbort(401, "AuthFailure", "Invalid credentials.")
}

// CheckEncryptPassword checks and encrypts the password in the user
// object.
func (u *User) CheckEncryptPassword(api *Api) error {

	// if empty or redacted check to see if we have an
	// existing user record
	if u.Password == "" || u.Password == RedactMsg {
		code, existingUser, err := api.GetUser(u.Id)
		if err != nil {
			return err
		}

		if code == 200 {
			// user has a password, assign it
			u.Password = existingUser.Source.Password
			return nil
		}

		if code >= 500 {
			return errors.New("bad response from Es while looking up user")
		}
	}

	// check the password
	if len(u.Password) < 10 {
		return errors.New("password must be over ten characters")
	}

	// encrypt the password
	// hash the password
	encPw, err := bcrypt.GenerateFromPassword([]byte(u.Password), EncCost)
	if err != nil {
		return err
	}

	// set the hashed password
	u.Password = string(encPw)

	return nil
}

// GetUserMapping
func GetUserMapping(prefix string) es.IndexTemplate {
	template := es.Obj{
		"index_patterns": []string{prefix + IdxUser},
		"settings": es.Obj{
			"number_of_shards": 5,
		},
		"mappings": es.Obj{
			"_doc": es.Obj{
				"_source": es.Obj{
					"enabled": true,
				},
				"properties": es.Obj{
					"id": es.Obj{
						"type": "text",
					},
					"description": es.Obj{
						"type": "text",
					},
					"display_name": es.Obj{
						"type": "text",
					},
					"active": es.Obj{
						"type": "boolean",
					},
					"sysop": es.Obj{
						"type": "boolean",
					},
					"password": es.Obj{
						"type": "keyword",
					},
					"sections_all": es.Obj{
						"type": "boolean",
					},
					"accounts": es.Obj{
						"type": "keyword",
					},
					"admin_accounts": es.Obj{
						"type": "keyword",
					},
				},
			},
		},
	}

	return es.IndexTemplate{
		Name:     prefix + IdxUser,
		Template: template,
	}
}
