/*
   Copyright 2019 txn2

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/
package provision

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/txn2/ack"
	"github.com/txn2/es"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const IdxAccount = "account"

// AccessKey
type AccessKey struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Key         string `json:"key"`
	Active      bool   `json:"active"`
}

// User defines a DCP user object
type Account struct {
	Id          string      `json:"id"`
	Description string      `json:"description"`
	DisplayName string      `json:"display_name"`
	Active      bool        `json:"active"`
	Modules     []string    `json:"modules"`
	OrgId       int         `json:"org_id"`
	AccessKeys  []AccessKey `json:"access_keys"`
}

// AccountResult returned from Elastic
type AccountResult struct {
	es.Result
	Source Account `json:"_source"`
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

	code, esResult, err := a.UpsertAccount(account)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		ak.SetPayloadType("EsError")
		ak.SetPayload("Error communicating with database.")
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
func (a *Api) UpsertAccount(account *Account) (int, es.Result, error) {
	a.Logger.Info("Upsert account record", zap.String("id", account.Id), zap.String("display_name", account.DisplayName))

	code, accountRes, _ := a.GetAccount(account.Id)
	if code == 200 {
		account.OrgId = accountRes.Source.OrgId
	}

	// attempt to encrypt the keys if one or more we provided
	// otherwise populate with existing
	err := account.CheckEncryptKeys(a)
	if err != nil {
		return 500, es.Result{}, err
	}

	return a.Elastic.PutObj(fmt.Sprintf("%s/_doc/%s", a.IdxPrefix+IdxAccount, account.Id), account)
}

// GetAccount
func (a *Api) GetAccount(id string) (int, *AccountResult, error) {

	accountResult := &AccountResult{}

	code, ret, err := a.Elastic.Get(fmt.Sprintf("%s/_doc/%s", a.IdxPrefix+IdxAccount, id))
	if err != nil {
		return code, accountResult, err
	}

	err = json.Unmarshal(ret, accountResult)
	if err != nil {
		return code, accountResult, err
	}

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
	code, existingAccount, err := api.GetAccount(acnt.Id)
	if err != nil {
		return err
	}

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
