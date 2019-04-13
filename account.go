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
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/txn2/ack"
	"github.com/txn2/es"
	"go.uber.org/zap"
)

const IxdAccount = "account"

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

	return a.Elastic.PutObj(fmt.Sprintf("%s/_doc/%s", a.IdxPrefix+IxdAccount, account.Id), account)
}

// GetAccount
func (a *Api) GetAccount(id string) (int, *AccountResult, error) {

	accountResult := &AccountResult{}

	code, ret, err := a.Elastic.Get(fmt.Sprintf("%s/_doc/%s", a.IdxPrefix+IxdAccount, id))
	if err != nil {
		return code, accountResult, err
	}

	err = json.Unmarshal(ret, accountResult)
	if err != nil {
		return code, accountResult, err
	}

	return code, accountResult, nil
}
