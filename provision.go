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
	"github.com/txn2/ack"
	"github.com/txn2/es"
	"go.uber.org/zap"
)

// Config
type Config struct {
	Logger     *zap.Logger
	HttpClient *ack.Client

	// used for communication with Elasticsearch
	// if nil, one will be created
	Elastic       *es.Client
	ElasticServer string

	// used to prefix the user and account indexes IdxPrefix_user, IdxPrefix_account
	// defaults to system.
	IdxPrefix string
}

// Api
type Api struct {
	*Config
}

// NewApi
func NewApi(cfg *Config) *Api {
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

	return a
}
