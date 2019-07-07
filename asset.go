package provision

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/txn2/ack"
	"github.com/txn2/es/v2"
	"go.uber.org/zap"
)

const IdxAsset = "asset"

// Condition
type ConditionCfg struct {
	Parser    string `json:"parser"`
	Condition string `json:"condition"`
}

// Route
type Route struct {
	AccountId string `json:"account_id" yaml:"accountId"`
	ModelId   string `json:"model_id" yaml:"modelId"`

	// system or account
	Type string `json:"type" yaml:"type"`

	// conditional routing used by
	// edge parsers (qlrx, etc)
	Conditions []ConditionCfg `json:"conditions" yaml:"conditions"`
}

// Asset defines an asset object
type Asset struct {
	Id          string  `json:"id" yaml:"id"`
	AccountId   string  `json:"account_id" yaml:"accountId"`
	Description string  `json:"description" yaml:"description"`
	DisplayName string  `json:"display_name" yaml:"displayName"`
	AssetClass  string  `json:"asset_class" yaml:"assetClass"`
	AssetCfg    string  `json:"asset_cfg" yaml:"assetCfg"`
	Active      bool    `json:"active" yaml:"active"`
	Routes      []Route `json:"routes" yaml:"routes"`
}

// AssetResult returned from Elastic
type AssetResult struct {
	es.Result
	Source Asset `json:"_source"`
}

// AssetResultAck
type AssetResultAck struct {
	ack.Ack
	Payload AssetResult `json:"payload"`
}

// AssetSummaryResult
type AssetSummaryResult struct {
	es.Result
	Source Account `json:"_source"`
}

// AccountSummary
type AssetSummary struct {
	Id          string   `json:"id" yaml:"id"`
	DisplayName string   `json:"display_name" yaml:"displayName"`
	Description string   `json:"description" yaml:"description"`
	Active      bool     `json:"active" yaml:"active"`
	Modules     []string `json:"modules" yaml:"modules"`
}

// AccountSummaryResults
type AssetSummaryResults struct {
	es.SearchResults
	Hits struct {
		Total    int                  `json:"total"`
		MaxScore float64              `json:"max_score"`
		Hits     []AssetSummaryResult `json:"hits"`
	} `json:"hits"`
}

// GetAdmAssetsHandler
func (a *Api) AssetAdmAssocHandler(c *gin.Context) {
	ak := ack.Gin(c)

	parentAccountId := c.Param("parentAccount")
	assetId := c.Param("asset")
	accountFromId := c.Param("accountFrom")
	accountToId := c.Param("accountTo")

	a.Logger.Info("Re-associate asset attempt.",
		zap.String("parentAccount", parentAccountId),
		zap.String("accountFrom", accountFromId),
		zap.String("accountTo", accountToId))

	if parentAccountId != accountFromId {
		// get account from
		code, accountFrom, err := a.GetAccount(accountFromId)
		if err != nil {
			a.Logger.Error("GetAdmAccountError", zap.Int("code", code), zap.Error(err))
			ak.SetPayloadType("GetAdmAccountError")
			ak.SetPayload("Error communicating with database.")
			ak.GinErrorAbort(code, "EsError", err.Error())
			return
		}

		if parentAccountId != accountFrom.Source.Parent {
			ak.SetPayload("From account is not a child of requester (or the requestor).")
			ak.GinErrorAbort(code, "AccountAccessError", ak.Ack.Payload.(string))
			return
		}
	}

	if parentAccountId != accountToId {
		// get account to
		code, accountTo, err := a.GetAccount(accountToId)
		if err != nil {
			a.Logger.Error("GetAdmAccountError", zap.Int("code", code), zap.Error(err))
			ak.SetPayloadType("GetAdmAccountError")
			ak.SetPayload("Error communicating with database.")
			ak.GinErrorAbort(code, "EsError", err.Error())
			return
		}

		if parentAccountId != accountTo.Source.Parent {
			ak.SetPayload("To account is not a child of requester (or the requestor).")
			ak.GinErrorAbort(code, "AccountAccessError", ak.Ack.Payload.(string))
			return
		}
	}

	// get the asset and requested
	code, assetResult, err := a.GetAsset(assetId)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		ak.SetPayloadType("EsError")
		ak.SetPayload("Error communicating with database.")
		ak.GinErrorAbort(500, "EsError", err.Error())
		return
	}

	if code >= 400 && code < 500 {
		ak.SetPayload("Asset " + assetId + " not found.")
		ak.GinErrorAbort(404, "AssetNotFound", "Asset not found")
		return
	}

	ii := 0

	for i, rt := range assetResult.Source.Routes {
		a.Logger.Info("Route", zap.String("account_id", rt.AccountId))
		if rt.AccountId == accountFromId {
			ii++
			assetResult.Source.Routes[i].AccountId = accountToId
		}
	}

	if ii < 1 {
		ak.SetPayloadType("AssociationError")
		ak.SetPayload("No routes to re-associate.")
		ak.GinErrorAbort(404, "NoAssociated", ak.Ack.Payload.(string))
		return
	}

	code, esResult, errorResponse, err := a.UpsertAsset(&assetResult.Source)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		ak.SetPayloadType("EsError")
		ak.SetPayload("Error communicating with database.")
		if errorResponse != nil {
			a.Logger.Error("EsErrorResponse", zap.String("es_error_response", errorResponse.Message))
			ak.SetPayload(errorResponse)
		}
		ak.GinErrorAbort(500, "EsError", err.Error())
		return
	}

	ak.SetPayloadType("Result")
	ak.GinSend(esResult)
}

// GetAdmAssetsHandler
// is same as parentAccount or account is a child of parentAccount
func (a *Api) GetAdmAssetsHandler(c *gin.Context) {
	ak := ack.Gin(c)

	// @todo check that parent has permission to access
	// child or is the same
	//parentAccountId := c.Param("parentAccount")
	accountId := c.Param("account")

	code, esResult, errorResponse, err := a.AssetAdmAssoc(accountId)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		ak.SetPayloadType("EsError")
		ak.SetPayload("Error communicating with database.")
		if errorResponse != nil {
			a.Logger.Error("EsErrorResponse", zap.String("es_error_response", errorResponse.Message))
			ak.SetPayload(errorResponse)
		}
		ak.GinErrorAbort(500, "EsError", err.Error())
		return
	}

	if code >= 400 && code < 500 {
		ak.SetPayload(errorResponse)
		if errorResponse != nil {
			a.Logger.Error("EsErrorResponse", zap.String("es_error_response", errorResponse.Message))
			ak.SetPayload(errorResponse)
		}
		ak.GinErrorAbort(code, "SearchError", "There was a problem searching")
		return
	}

	ak.SetPayloadType("AssetSummaryResults")
	ak.GinSend(esResult)
}

// AssetAdmAssoc
func (a *Api) AssetAdmAssoc(accountId string) (int, AssetSummaryResults, *es.ErrorResponse, error) {
	query := es.Obj{
		"query": es.Obj{
			"nested": es.Obj{
				"path": "routes",
				"query": es.Obj{
					"bool": es.Obj{
						"must": []es.Obj{
							{
								"match": es.Obj{"routes.account_id": accountId},
							},
						},
					},
				},
			},
		},
		"sort": es.Obj{
			"id": "asc",
		},
	}

	asResults := &AssetSummaryResults{}

	code, errorResponse, err := a.Elastic.PostObjUnmarshal(fmt.Sprintf("%s/_search", a.IdxPrefix+IdxAsset), query, asResults)
	if err != nil {
		return code, *asResults, errorResponse, err
	}

	return code, *asResults, errorResponse, nil
}

// UpsertAssetHandler
func (a *Api) UpsertAssetHandler(c *gin.Context) {
	ak := ack.Gin(c)

	asset := &Asset{}
	err := ak.UnmarshalPostAbort(asset)
	if err != nil {
		a.Logger.Error("Upsert failure.", zap.Error(err))
		return
	}

	code, esResult, errorResponse, err := a.UpsertAsset(asset)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		ak.SetPayloadType("EsError")
		ak.SetPayload("Error communicating with database.")
		if errorResponse != nil {
			a.Logger.Error("EsErrorResponse", zap.String("es_error_response", errorResponse.Message))
			ak.SetPayload(errorResponse)
		}
		ak.GinErrorAbort(500, "EsError", err.Error())
		return
	}

	if code < 200 || code >= 300 {
		a.Logger.Error("Es returned a non 200")
		ak.SetPayload(esResult)
		ak.GinErrorAbort(500, "EsError", "Es returned a non 200")
		return
	}

	ak.SetPayloadType("EsResult")
	ak.GinSend(esResult)
}

// UpsertAccount inserts or updates an asset. Elasticsearch
// treats documents as immutable.
func (a *Api) UpsertAsset(asset *Asset) (int, es.Result, *es.ErrorResponse, error) {
	a.Logger.Info("Upsert asset record", zap.String("asset_id", asset.Id), zap.String("display_name", asset.DisplayName))

	return a.Elastic.PutObj(fmt.Sprintf("%s/_doc/%s", a.IdxPrefix+IdxAsset, asset.Id), asset)
}

// GetAsset
func (a *Api) GetAsset(id string) (int, *AssetResult, error) {

	assetResult := &AssetResult{}

	code, ret, err := a.Elastic.Get(fmt.Sprintf("%s/_doc/%s", a.IdxPrefix+IdxAsset, id))
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		return code, assetResult, err
	}

	err = json.Unmarshal(ret, assetResult)
	if err != nil {
		return code, assetResult, err
	}

	return code, assetResult, nil
}

// GetAssetHandler gets an asset by ID
func (a *Api) GetAssetHandler(c *gin.Context) {
	ak := ack.Gin(c)

	id := c.Param("id")
	code, assetResult, err := a.GetAsset(id)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		ak.SetPayloadType("EsError")
		ak.SetPayload("Error communicating with database.")
		ak.GinErrorAbort(500, "EsError", err.Error())
		return
	}

	if code >= 400 && code < 500 {
		ak.SetPayload("Asset " + id + " not found.")
		ak.GinErrorAbort(404, "AssetNotFound", "Asset not found")
		return
	}

	ak.SetPayloadType("AssetResult")
	ak.GinSend(assetResult)
}

// GetAssetMapping
func GetAssetMapping(prefix string) es.IndexTemplate {
	template := es.Obj{
		"index_patterns": []string{prefix + IdxAsset},
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
						"type": "keyword",
					},
					"account_id": es.Obj{
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
					"asset_class": es.Obj{
						"type": "keyword",
					},
					"asset_cfg": es.Obj{
						"type": "text",
					},
					"routes": es.Obj{
						"type": "nested",
						"properties": es.Obj{
							"account_id": es.Obj{
								"type": "keyword",
							},
							"model_id": es.Obj{
								"type": "keyword",
							},
							"type": es.Obj{
								"type": "keyword",
							},
							"conditions": es.Obj{
								"type": "nested",
								"properties": es.Obj{
									"parser": es.Obj{
										"type": "keyword",
									},
									"condition": es.Obj{
										"type": "text",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return es.IndexTemplate{
		Name:     prefix + IdxAsset,
		Template: template,
	}
}
