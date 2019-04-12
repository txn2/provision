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
	"github.com/gin-gonic/gin"
	"github.com/txn2/ack"
)

func main() {

	server := ack.NewServer()


	// Upsert an account
	server.Router.POST("/account", func(c *gin.Context) {
		ak := ack.Gin(c)
		ak.SetPayloadType("Message")
		ak.GinSend("Upsert account.")
	})

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