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

const IdxAccount = "account"

// AccessKey
type AccessKey struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Key         string `json:"key" yaml:"key"`
	Active      bool   `json:"active" yaml:"active"`
}

// Account defines an account object
type Account struct {
	Id          string      `json:"id" yaml:"id"`
	Parent      string      `json:"parent" yaml:"parent"`
	Description string      `json:"description" yaml:"description"`
	DisplayName string      `json:"display_name" yaml:"displayName"`
	Active      bool        `json:"active" yaml:"active"`
	Modules     []string    `json:"modules" yaml:"modules"`
	OrgId       int         `json:"org_id" yaml:"orgId"`
	AccessKeys  []AccessKey `json:"access_keys" yaml:"accessKeys"`
}

// AccountResult returned from Elastic
type AccountResult struct {
	es.Result
	Source Account `json:"_source"`
}

// AccountResultAck
type AccountResultAck struct {
	ack.Ack
	Payload AccountResult `json:"payload"`
}

// AccountResult
type AccountSummaryResult struct {
	es.Result
	Source AccountSummary `json:"_source"`
}

// AccountSummary
type AccountSummary struct {
	Id          string   `json:"id" yaml:"id"`
	DisplayName string   `json:"display_name" yaml:"displayName"`
	Description string   `json:"description" yaml:"description"`
	Active      bool     `json:"active" yaml:"active"`
	Modules     []string `json:"modules" yaml:"modules"`
}

// AccountSummaryResults
type AccountSummaryResults struct {
	es.SearchResults
	Hits struct {
		Total    int                    `json:"total"`
		MaxScore float64                `json:"max_score"`
		Hits     []AccountSummaryResult `json:"hits"`
	} `json:"hits"`
}

// GetAdmChildAccounts
// get a list of account with a parent account id
func (a *Api) GetAdmChildAccounts(accountId string) (int, AccountSummaryResults, *es.ErrorResponse, error) {
	query := es.Obj{
		"_source": []string{"id", "display_name", "description", "active", "modules"},
		"query": es.Obj{
			"constant_score": es.Obj{
				"filter": es.Obj{
					"term": es.Obj{
						"parent": accountId,
					},
				},
			},
		},
	}

	js, _ := json.Marshal(query)

	pth := fmt.Sprintf("%s/_search", a.IdxPrefix+IdxAccount)

	a.Logger.Info("Searching", zap.String("url", pth))
	a.Logger.Info("Query", zap.ByteString("json", js))

	asResults := &AccountSummaryResults{}

	code, errorResponse, err := a.Elastic.PostObjUnmarshal(pth, query, asResults)
	if err != nil {
		return code, *asResults, errorResponse, err
	}

	return code, *asResults, nil, nil
}

// GetAdmChildAccountsHandler
func (a *Api) GetAdmChildAccountsHandler(c *gin.Context) {
	ak := ack.Gin(c)

	parentAccountId := c.Param("parentAccount")

	code, esResult, errorResponse, err := a.GetAdmChildAccounts(parentAccountId)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		ak.SetPayloadType("EsError")
		ak.SetPayload("Error communicating with database.")
		if errorResponse != nil {
			ak.SetPayloadType("EsErrorResponse")
			ak.SetPayload(errorResponse)
		}
		ak.GinErrorAbort(500, "EsError", err.Error())
		return
	}

	if code >= 400 && code < 500 {
		ak.SetPayload(errorResponse)
		ak.GinErrorAbort(code, "SearchError", "There was a problem searching")
		return
	}

	ak.SetPayloadType("AccountSummaryResults")
	ak.GinSend(esResult)
}

// GetAdmAccountHandler
func (a *Api) GetAdmAccountHandler(c *gin.Context) {
	ak := ack.Gin(c)
	parentAccountId := c.Param("parentAccount")
	accountId := c.Param("account")

	code, account, err := a.GetAccount(accountId)
	if err != nil {
		a.Logger.Error("GetAdmAccountError", zap.Int("code", code), zap.Error(err))
		ak.SetPayloadType("GetAdmAccountError")
		ak.SetPayload("Error communicating with database.")
		ak.GinErrorAbort(code, "EsError", err.Error())
		return
	}

	// if the account is not looking up data
	// on it self then check to see if the child account
	// has the requesting account as a parent
	if parentAccountId != accountId && parentAccountId != account.Source.Parent {
		ak.SetPayload("Account requested is not a child of requester.")
		ak.GinErrorAbort(code, "AccountAccessError", ak.Ack.Payload.(string))
		return
	}

	ak.SetPayloadType("AccountResult")
	ak.GinSend(account)
}

// UpsertAccountHandler
func (a *Api) UpsertAdmChildAccountHandler(c *gin.Context) {
	ak := ack.Gin(c)

	account := &Account{}
	err := ak.UnmarshalPostAbort(account)
	if err != nil {
		a.Logger.Error("Upsert failure.", zap.Error(err))
		return
	}

	parentAccountId := c.Param("parentAccount")

	code, accountRes, _ := a.GetAccount(account.Id)
	if code == 200 {
		if accountRes.Source.Parent != parentAccountId {
			ak.SetPayloadType("ValidationError")
			ak.SetPayload("Account exists but does not belong to parent.")
			ak.GinErrorAbort(500, "ValidationError", "Account does not belong to parent.")
			return
		}
	}

	// Adm account must have the requesting account as the parent
	account.Parent = parentAccountId

	code, esResult, errorResonse, err := a.UpsertAccount(account)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		ak.SetPayloadType("EsError")
		ak.SetPayload("Error communicating with database.")
		if errorResonse != nil {
			ak.SetPayloadType("EsErrorResponse")
			ak.SetPayload(errorResonse)
		}
		ak.GinErrorAbort(500, "EsError", err.Error())
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

// UpsertAccountHandler
func (a *Api) UpsertAccountHandler(c *gin.Context) {
	ak := ack.Gin(c)

	account := &Account{}
	err := ak.UnmarshalPostAbort(account)
	if err != nil {
		a.Logger.Error("Upsert failure.", zap.Error(err))
		return
	}

	code, esResult, errorResponse, err := a.UpsertAccount(account)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		ak.SetPayloadType("EsError")
		ak.SetPayload("Error communicating with database.")
		if errorResponse != nil {
			a.Logger.Error("EsErrorResponse", zap.String("es_error_response", errorResponse.Message))
			ak.SetPayloadType("EsErrorResponse")
			ak.SetPayload(errorResponse)
		}
		ak.GinErrorAbort(500, "EsError", err.Error())
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

// UpsertAccount inserts or updates an account. Elasticsearch
// treats documents as immutable.
func (a *Api) UpsertAccount(account *Account) (int, es.Result, *es.ErrorResponse, error) {
	a.Logger.Info("Upsert account record", zap.String("id", account.Id), zap.String("display_name", account.DisplayName))

	code, accountRes, _ := a.GetAccount(account.Id)
	if code == 200 {
		account.OrgId = accountRes.Source.OrgId
	}

	// attempt to encrypt the keys if one or more were provided
	// otherwise populate with existing
	err := account.CheckEncryptKeys(a)
	if err != nil {
		return 500, es.Result{}, nil, err
	}

	return a.Elastic.PutObj(fmt.Sprintf("%s/_doc/%s", a.IdxPrefix+IdxAccount, account.Id), account)
}

// CheckKeyHandler
func (a *Api) CheckKeyHandler(c *gin.Context) {
	ak := ack.Gin(c)

	accountId := c.Param("id")
	accessKey := &AccessKey{}
	err := ak.UnmarshalPostAbort(accessKey)
	if err != nil {
		a.Logger.Error("Key failure.", zap.Error(err))
		return
	}

	ok, err := a.CheckKey(accountId, *accessKey)
	if err != nil {
		ak.SetPayload("Access key check failure.")
		ak.GinErrorAbort(404, "CheckKeyFailed", err.Error())
		return
	}

	ak.SetPayloadType("CheckKeyResult")

	if ok {
		ak.GinSend(true)
		return
	}

	ak.SetPayload(false)
	ak.GinErrorAbort(401, "CheckKeyFailed", "Key is not valid for account.")
}

// CheckKey returns true if the provided key is valid for the account
func (a *Api) CheckKey(accountId string, key AccessKey) (bool, error) {
	// Get the requested account
	code, accountResult, err := a.getAccountRaw(accountId)
	if err != nil {
		return false, err
	}

	if code != 200 {
		return false, errors.New("Got status code " + string(code) + " back from GetAccount.")
	}

	for _, accessKey := range accountResult.Source.AccessKeys {
		// if we find an active key with the same name
		if accessKey.Name == key.Name && accessKey.Active == true {
			// check key (password)
			err = bcrypt.CompareHashAndPassword([]byte(accessKey.Key), []byte(key.Key))
			if err != nil {
				return false, nil
			}
			return true, nil
		}
	}

	return false, nil
}

// getAccountRaw returns raw account (un-redacted)
func (a *Api) getAccountRaw(id string) (int, *AccountResult, error) {

	accountResult := &AccountResult{}

	code, ret, err := a.Elastic.Get(fmt.Sprintf("%s/_doc/%s", a.IdxPrefix+IdxAccount, id))
	if err != nil {
		return code, accountResult, err
	}

	//a.Logger.Info("About to unmarshal", zap.Int("code", code), zap.ByteString("data", ret))

	if code == 200 {
		err = json.Unmarshal(ret, accountResult)
		if err != nil {
			return code, accountResult, err
		}

		return code, accountResult, nil
	}

	return code, nil, fmt.Errorf("database returned code %d:%s", code, ret)
}

// GetAccount
func (a *Api) GetAccount(id string) (int, *AccountResult, error) {

	code, accountResult, err := a.getAccountRaw(id)
	if err != nil || code != 200 {
		return code, nil, err
	}

	// Redact keys
	for i := range accountResult.Source.AccessKeys {
		accountResult.Source.AccessKeys[i].Key = RedactMsg
	}

	return code, accountResult, nil
}

// GetAccountHandler gets an account by ID
func (a *Api) GetAccountHandler(c *gin.Context) {
	ak := ack.Gin(c)

	id := c.Param("id")
	code, accountResult, err := a.GetAccount(id)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		ak.SetPayloadType("EsError")
		ak.SetPayload("Error communicating with database.")
		ak.GinErrorAbort(500, "EsError", err.Error())
		return
	}

	if code >= 400 && code < 500 {
		ak.SetPayload("Account " + id + " not found.")
		ak.GinErrorAbort(404, "AccountNotFound", "Account not found")
		return
	}

	ak.SetPayloadType("AccountResult")
	ak.GinSend(accountResult)
}

// CheckEncryptKeys checks and encrypts keys in the account
// object.
func (acnt *Account) CheckEncryptKeys(api *Api) error {

	// does the account exist?
	code, existingAccount, _ := api.GetAccount(acnt.Id)

	// account exists
	if code == 200 {
		// assign existing encrypted keys if current data is
		// empty or redacted message
		for i, accessKey := range acnt.AccessKeys {
			// empty or redacted keys mean use existing
			if accessKey.Key == "" || accessKey.Key == RedactMsg {
				acnt.AccessKeys[i].Key = existingAccount.Source.AccessKeys[i].Key
				continue
			}

			// check the key length
			if len(accessKey.Key) < 10 {
				return errors.New("key must be over ten characters")
			}

			// encrypt the key
			encKey, err := bcrypt.GenerateFromPassword([]byte(accessKey.Key), EncCost)
			if err != nil {
				return err
			}

			// set the hashed password
			acnt.AccessKeys[i].Key = string(encKey)
		}

		return nil
	}

	if code >= 500 {
		return errors.New("bad response from Es while looking up account")
	}

	// if we got here this is a new account
	// check for key lengths and encrypt the keys
	for i, accessKey := range acnt.AccessKeys {
		if len(accessKey.Key) < 10 {
			return errors.New("key must be over ten characters")
		}

		encKey, err := bcrypt.GenerateFromPassword([]byte(accessKey.Key), EncCost)
		if err != nil {
			return err
		}

		// set the hashed password
		acnt.AccessKeys[i].Key = string(encKey)
	}

	return nil
}

// GetAccountMapping
func GetAccountMapping(prefix string) es.IndexTemplate {
	template := es.Obj{
		"index_patterns": []string{prefix + IdxAccount},
		"settings": es.Obj{
			"number_of_shards": 2,
		},
		"mappings": es.Obj{
			"_doc": es.Obj{
				"_source": es.Obj{
					"enabled": true,
				},
				"properties": es.Obj{
					"id": es.Obj{
						"type": "keyword",
					},
					"parent": es.Obj{
						"type": "keyword",
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
					"org_id": es.Obj{
						"type": "keyword",
					},
					"modules": es.Obj{
						"type": "keyword",
					},
					"access_keys": es.Obj{
						"type": "nested",
						"properties": es.Obj{
							"name": es.Obj{
								"type": "text",
							},
							"description": es.Obj{
								"type": "text",
							},
							"key": es.Obj{
								"type": "keyword",
							},
							"active": es.Obj{
								"type": "boolean",
							},
						},
					},
				},
			},
		},
	}

	return es.IndexTemplate{
		Name:     prefix + IdxAccount,
		Template: template,
	}
}
