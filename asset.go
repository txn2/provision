package provision

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/txn2/ack"
	"github.com/txn2/es"
	"go.uber.org/zap"
)

const IdxAsset = "asset"

// AccountModel
type AccountModel struct {
	AccountId string `json:"account_id"`
	ModelId   string `json:"model_id"`
}

// Asset defines an asset object
type Asset struct {
	Id            string         `json:"id"`
	Description   string         `json:"description"`
	DisplayName   string         `json:"display_name"`
	AssetClass    string         `json:"asset_class"`
	AssetCfg      string         `json:"asset_cfg"`
	Active        bool           `json:"active"`
	AccountModels []AccountModel `json:"account_models"`
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

// UpsertAssetHandler
func (a *Api) UpsertAssetHandler(c *gin.Context) {
	ak := ack.Gin(c)

	asset := &Asset{}
	err := ak.UnmarshalPostAbort(asset)
	if err != nil {
		a.Logger.Error("Upsert failure.", zap.Error(err))
		return
	}

	code, esResult, err := a.UpsertAsset(asset)
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

// UpsertAccount inserts or updates an asset. Elasticsearch
// treats documents as immutable.
func (a *Api) UpsertAsset(asset *Asset) (int, es.Result, error) {
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
					"asset_class": es.Obj{
						"type": "keyword",
					},
					"asset_cfg": es.Obj{
						"type": "text",
					},
					"account_models": es.Obj{
						"properties": es.Obj{
							"account_id": es.Obj{
								"type": "keyword",
							},
							"model_id": es.Obj{
								"type": "keyword",
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
