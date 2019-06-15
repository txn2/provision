package main

import (
	"flag"
	"os"

	"github.com/txn2/micro"
	"github.com/txn2/provision"
)

var (
	elasticServerEnv = getEnv("ELASTIC_SERVER", "http://elasticsearch:9200")
	systemPrefixEnv  = getEnv("SYSTEM_PREFIX", "system_")
)

func main() {

	esServer := flag.String("esServer", elasticServerEnv, "Elasticsearch Server")
	systemPrefix := flag.String("systemPrefix", systemPrefixEnv, "Prefix for system indices.")

	serverCfg, _ := micro.NewServerCfg("Provision")
	server := micro.NewServer(serverCfg)

	// Provision API
	provApi, err := provision.NewApi(&provision.Config{
		Logger:        server.Logger,
		HttpClient:    server.Client,
		ElasticServer: *esServer,
		IdxPrefix:     *systemPrefix,
		Token:         server.Token,
	})
	if err != nil {
		server.Logger.Fatal("failure to instantiate the provisioning API: " + err.Error())
		os.Exit(1)
	}

	// system prefix
	server.Router.GET("/prefix", provApi.PrefixHandler)

	// Upsert an account
	server.Router.POST("/account", provApi.UpsertAccountHandler)

	// Get an account
	server.Router.GET("/account/:id", provApi.GetAccountHandler)

	// Check an account for an active key
	server.Router.POST("/keyCheck/:id", provApi.CheckKeyHandler)

	// Search accounts
	server.Router.POST("/searchAccounts", provApi.SearchAccountsHandler)

	// Upsert a user
	server.Router.POST("/user", provApi.UpsertUserHandler)

	// Get a user
	server.Router.GET("/user/:id", provApi.GetUserHandler)

	// Search users
	server.Router.POST("/searchUsers", provApi.SearchUsersHandler)

	// User has basic access (checks token and access request object)
	server.Router.POST("/userHasAccess", provision.UserTokenHandler(), provision.UserHasAccessHandler)

	// User has admin access (checks token and access request object)
	server.Router.POST("/userHasAdminAccess", provision.UserTokenHandler(), provision.UserHasAdminAccessHandler)

	// Auth a user
	server.Router.POST("/authUser", provApi.AuthUserHandler)

	// Upsert an asset
	server.Router.POST("/asset", provApi.UpsertAssetHandler)

	// Get an asset
	server.Router.GET("/asset/:id", provApi.GetAssetHandler)

	// Search assets
	server.Router.POST("/searchAssets", provApi.SearchAssetsHandler)

	// Account Admin Routes
	// use internally with no authentication or
	// use through the adm proxy externally which validates
	// access to :parentAccount
	adm := server.Router.Group("/adm/:parentAccount")

	// Get Account
	// must be the same account or a child
	adm.GET("/account/:account", provApi.GetAdmAccountHandler)

	// Upsert a child account
	adm.POST("/account", provApi.UpsertAdmChildAccountHandler)

	// Get Children
	adm.GET("/children", provApi.GetAdmChildAccountsHandler)

	// Get asset associations for :account
	adm.GET("/assets/:account", provApi.GetAdmAssetsHandler)

	// Asset Re-Association
	// re-associate an asset with an account and model
	// an existing association to the parent or one of the
	// children is required.
	// @todo implement
	adm.POST("/assetAssoc", provApi.AssetAdmAssocHandler)

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
