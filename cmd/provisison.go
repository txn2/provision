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
package main

import (
	"flag"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/txn2/ack"
	"github.com/txn2/provision"
)

var (
	elasticServerEnv = getEnv("ELASTIC_SERVER", "http://elasticsearch:9200")
)

func main() {

	esServer := flag.String("esServer", elasticServerEnv, "Elasticsearch Server")

	server := ack.NewServer()

	// Provision API
	provApi := provision.NewApi(&provision.Config{
		Logger:        server.Logger, // pre configured zap logger
		HttpClient:    server.Client,
		ElasticServer: *esServer,
	})

	// Upsert an account
	server.Router.POST("/account", provApi.UpsertAccountHandler)

	// Get an account
	server.Router.GET("/account/:id", func(c *gin.Context) {
		ak := ack.Gin(c)
		ak.SetPayloadType("Message")
		ak.GinSend("Get an account.")
	})

	// Upsert a user
	server.Router.POST("/user", func(c *gin.Context) {
		ak := ack.Gin(c)
		ak.SetPayloadType("Message")
		ak.GinSend("Upsert user.")
	})

	// Get a user
	server.Router.GET("/user/:id", func(c *gin.Context) {
		ak := ack.Gin(c)
		ak.SetPayloadType("Message")
		ak.GinSend("Upsert user.")
	})

	// run provisioning server
	server.Run()
}

// getEnv gets an environment variable or sets a default if
// one does not exist.
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}

	return value
}
