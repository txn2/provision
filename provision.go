// Copyright 2019 txn2
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package provision implements Account, User and Asset objects for use
// in txn2 projects.
package provision

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/txn2/ack"
	"github.com/txn2/es/v2"
	"github.com/txn2/micro"
	"github.com/txn2/token"
	"go.uber.org/zap"
)

// Config
type Config struct {
	Logger     *zap.Logger
	HttpClient *micro.Client

	// used for communication with Elasticsearch
	// if nil, one will be created
	Elastic       *es.Client
	ElasticServer string

	// used to prefix the user and account indexes IdxPrefix_user, IdxPrefix_account
	// defaults to system.
	IdxPrefix string

	// pre-configured from server (txn2/micro)
	Token *token.Jwt
}

// Api
type Api struct {
	*Config
}

// NewApi
func NewApi(cfg *Config) (*Api, error) {
	a := &Api{Config: cfg}

	if a.Elastic == nil {
		// Configure an elastic client
		a.Elastic = es.CreateClient(es.Config{
			Log:           cfg.Logger,
			HttpClient:    cfg.HttpClient.Http,
			ElasticServer: cfg.ElasticServer,
		})
	}

	if cfg.IdxPrefix == "" {
		cfg.IdxPrefix = "system_"
	}

	// check for elasticsearch a few times before failing
	// this reduces a reliance on restarts when a full system is
	// spinning up
	backOff := []int{10, 10, 15, 15, 30, 30, 45}
	for _, boff := range backOff {
		code, _, _ := a.Elastic.Get("")
		a.Logger.Info("Attempting to contact Elasticsearch", zap.String("server", a.Elastic.ElasticServer))

		if code == 200 {
			a.Logger.Info("Connection to Elastic search successful.", zap.String("server", a.Elastic.ElasticServer))
			break
		}

		a.Logger.Warn("Unable to contact Elasticsearch rolling back off.", zap.Int("wait_seconds", boff))
		<-time.After(time.Duration(boff) * time.Second)
	}

	// send index mappings for user
	err := a.SendEsMapping(GetUserMapping(cfg.IdxPrefix))
	if err != nil {
		return nil, err
	}

	// send index mappings for account
	err = a.SendEsMapping(GetAccountMapping(cfg.IdxPrefix))
	if err != nil {
		return nil, err
	}

	// send index mappings for asset
	err = a.SendEsMapping(GetAssetMapping(cfg.IdxPrefix))
	if err != nil {
		return nil, err
	}

	return a, nil
}

// SetupUserIndexTemplate
func (a *Api) SendEsMapping(mapping es.IndexTemplate) error {

	a.Logger.Info("Sending template",
		zap.String("type", "SendEsMapping"),
		zap.String("mapping", mapping.Name),
	)

	code, esResult, errorResponse, err := a.Elastic.PutObj(fmt.Sprintf("_template/%s", mapping.Name), mapping.Template)
	if err != nil {
		a.Logger.Error("Got error sending template", zap.Error(err))
		a.Logger.Error("EsErrorResponse", zap.String("es_error_response", errorResponse.Message))
		return err
	}

	if code != 200 {
		a.Logger.Error("Got code", zap.Int("code", code), zap.String("EsResult", esResult.ResultType))
		if errorResponse != nil {
			a.Logger.Error("EsErrorResponse", zap.String("es_error_response", errorResponse.Message))
		}
		return errors.New("Error setting up " + mapping.Name + " template, got code " + string(code))
	}

	return err
}

// PrefixHandler
func (a *Api) PrefixHandler(c *gin.Context) {
	ak := ack.Gin(c)
	ak.SetPayloadType("Prefix")
	ak.GinSend(a.Config.IdxPrefix)
}
