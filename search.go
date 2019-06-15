package provision

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/txn2/ack"
	"github.com/txn2/es/v2"
	"go.uber.org/zap"
)

// AssetSearchResults
type AssetSearchResults struct {
	es.SearchResults
	Hits struct {
		Total    int           `json:"total"`
		MaxScore float64       `json:"max_score"`
		Hits     []AssetResult `json:"hits"`
	} `json:"hits"`
}

// AssetSearchResultsAck
type AssetSearchResultsAck struct {
	ack.Ack
	Payload AssetSearchResults `json:"payload"`
}

// AccountSearchResults
type AccountSearchResults struct {
	es.SearchResults
	Hits struct {
		Total    int             `json:"total"`
		MaxScore float64         `json:"max_score"`
		Hits     []AccountResult `json:"hits"`
	} `json:"hits"`
}

// AccountSearchResultsAck
type AccountSearchResultsAck struct {
	ack.Ack
	Payload AccountSearchResults `json:"payload"`
}

// UserSearchResults
type UserSearchResults struct {
	es.SearchResults
	Hits struct {
		Total    int          `json:"total"`
		MaxScore float64      `json:"max_score"`
		Hits     []UserResult `json:"hits"`
	} `json:"hits"`
}

// UserSearchResultsAck
type UserSearchResultsAck struct {
	ack.Ack
	Payload UserSearchResults `json:"payload"`
}

// SearchAssets
func (a *Api) SearchAssets(searchObj *es.Obj) (int, AssetSearchResults, *es.ErrorResponse, error) {
	asResults := &AssetSearchResults{}

	code, errorResponse, err := a.Elastic.PostObjUnmarshal(fmt.Sprintf("%s/_search", a.IdxPrefix+IdxAsset), searchObj, asResults)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		return code, *asResults, errorResponse, err
	}

	return code, *asResults, nil, nil
}

// SearchAssetsHandler
func (a *Api) SearchAssetsHandler(c *gin.Context) {
	ak := ack.Gin(c)

	obj := &es.Obj{}
	err := ak.UnmarshalPostAbort(obj)
	if err != nil {
		a.Logger.Error("Search failure.", zap.Error(err))
		return
	}

	code, esResult, errorResponse, err := a.SearchAssets(obj)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		ak.SetPayloadType("EsError")
		ak.SetPayload("Error communicating with database.")
		if errorResponse != nil {
			a.Logger.Error("EsErrorResponse", zap.String("es_error_response", errorResponse.Message))
			ak.SetPayloadType("EsErrorResponse")
			ak.SetPayload(errorResponse)
		}
		ak.GinErrorAbort(code, "EsError", err.Error())
		return
	}

	if code >= 400 && code < 500 {
		ak.SetPayload(esResult)
		ak.GinErrorAbort(500, "SearchError", "There was a problem searching")
		return
	}

	ak.SetPayloadType("AssetSearchResults")
	ak.GinSend(esResult)
}

// SearchAccounts
func (a *Api) SearchAccounts(searchObj *es.Obj) (int, AccountSearchResults, *es.ErrorResponse, error) {
	asResults := &AccountSearchResults{}

	code, errorResponse, err := a.Elastic.PostObjUnmarshal(fmt.Sprintf("%s/_search", a.IdxPrefix+IdxAccount), searchObj, asResults)
	if err != nil {
		return code, *asResults, errorResponse, err
	}

	// Redact Keys
	for i := range asResults.Hits.Hits {
		for ii := range asResults.Hits.Hits[i].Source.AccessKeys {
			asResults.Hits.Hits[i].Source.AccessKeys[ii].Key = RedactMsg
		}
	}

	return code, *asResults, nil, nil
}

// SearchAccountsHandler
func (a *Api) SearchAccountsHandler(c *gin.Context) {
	ak := ack.Gin(c)

	obj := &es.Obj{}
	err := ak.UnmarshalPostAbort(obj)
	if err != nil {
		a.Logger.Error("Search failure.", zap.Error(err))
		return
	}

	code, esResult, errorResponse, err := a.SearchAccounts(obj)
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

	if code >= 400 && code < 500 {
		ak.SetPayload(esResult)
		ak.GinErrorAbort(500, "SearchError", "There was a problem searching")
		return
	}

	ak.SetPayloadType("AccountSearchResults")
	ak.GinSend(esResult)
}

// SearchUsers
func (a *Api) SearchUsers(searchObj *es.Obj) (int, UserSearchResults, *es.ErrorResponse, error) {
	usResults := &UserSearchResults{}

	code, errorResponse, err := a.Elastic.PostObjUnmarshal(fmt.Sprintf("%s/_search", a.IdxPrefix+IdxUser), searchObj, usResults)
	if err != nil {
		return code, *usResults, errorResponse, err
	}

	// Redact Passwords
	for i := range usResults.Hits.Hits {
		usResults.Hits.Hits[i].Source.Password = RedactMsg
	}

	return code, *usResults, nil, nil
}

// SearchUsersHandler
func (a *Api) SearchUsersHandler(c *gin.Context) {
	ak := ack.Gin(c)

	obj := &es.Obj{}
	err := ak.UnmarshalPostAbort(obj)
	if err != nil {
		a.Logger.Error("EsError", zap.Error(err))
		a.Logger.Error("Search failure.", zap.Error(err))
		return
	}

	code, esResult, errorResponse, err := a.SearchUsers(obj)
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

	if code >= 400 && code < 500 {
		ak.SetPayload(esResult)
		ak.GinErrorAbort(500, "SearchError", "There was a problem searching")
		return
	}

	ak.SetPayloadType("UserSearchResults")
	ak.GinSend(esResult)
}
