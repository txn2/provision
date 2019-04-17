package provision

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
	"github.com/txn2/ack"
	"github.com/txn2/token"
)

// AccessCheck is used to configure an access check
type AccessCheck struct {
	Sections []string `json:"sections"`
	Accounts []string `json:"accounts"`
}

// AccessCheckResult
type AccessCheckResult struct {
	AccessChecked *AccessCheck `json:"access_checked"`
	Status        bool         `json:"status"`
	Message       string       `json:"message"`
}

// AccountAccessCheckHandler
func AccountAccessCheckHandler(checkAdmin bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ak ack.GinAck

		userI, ok := c.Get("User")
		if !ok {
			ak = ack.Gin(c)
			ak.SetPayload("Unable to get user from token.")
			ak.GinErrorAbort(401, "E401", "UnauthorizedAccess")
			return
		}

		user := userI.(*User)

		account := c.Param("account")
		if account == "" {
			ak = ack.Gin(c)
			ak.SetPayload("No account specified.")
			ak.GinErrorAbort(401, "E401", "UnauthorizedAccess")
			return
		}

		ac := &AccessCheck{
			Accounts: []string{account},
		}

		if (!checkAdmin && user.HasAccess(ac)) || (checkAdmin && user.HasAdminAccess(ac)) {
			return
		}

		ak = ack.Gin(c)
		ak.SetPayload("User does not have required access.")
		ak.GinErrorAbort(401, "E401", "UnauthorizedAccess")
	}
}

// UserTokenHandler
func UserTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ak := ack.Gin(c)

		tokI, ok := c.Get("Tok")
		if !ok {
			ak.SetPayloadType("ErrorMessage")
			ak.SetPayload("missing token")
			ak.GinErrorAbort(401, "E401", "UnauthorizedAccess")
			return
		}

		tok := tokI.(*token.Tok)

		if !tok.Valid {
			ak.SetPayloadType("ErrorMessage")
			ak.SetPayload("invalid token")
			ak.GinErrorAbort(401, "E401", "UnauthorizedAccess")
			return
		}

		// check for expiration
		time.Local = time.UTC
		exp := int64(tok.Claims["exp"].(float64))
		if time.Now().Unix() > exp {
			ak.SetPayloadType("ErrorMessage")
			ak.SetPayload("expired token")
			ak.GinErrorAbort(401, "E401", "UnauthorizedAccess")
			return
		}

		user := &User{}
		err := mapstructure.Decode(tok.Claims["data"].(map[string]interface{}), user)
		if err != nil {
			ak.SetPayloadType("ErrorMessage")
			ak.SetPayload("there was a problem decoding the token")
			ak.GinErrorAbort(401, "E401", "UnauthorizedAccess")
			return
		}

		if !user.HasBasicAccess() {
			ak.SetPayloadType("ErrorMessage")
			ak.SetPayload("user does not have basic access")
			ak.GinErrorAbort(401, "E401", "UnauthorizedAccess")
			return
		}

		// set a user middleware
		c.Set("User", user)
	}
}

// UserHasAdminAccessHandler
func UserHasAdminAccessHandler(c *gin.Context) {
	c.Set("AdminCheck", true)
	UserHasAccessHandler(c)
}

// UserHasAccessHandler
func UserHasAccessHandler(c *gin.Context) {
	ak := ack.Gin(c)
	ak.SetPayloadType("AccessCheckResult")
	acr := AccessCheckResult{
		Status:  false,
		Message: "Filed to check status",
	}

	userI, ok := c.Get("User")
	if !ok {
		ak.SetPayloadType("AccessCheckResult")
		acr.Message = "No user object in token."
		ak.SetPayload(acr)
		ak.GinErrorAbort(401, "E401", "UnauthorizedAccess")
		return
	}

	user := userI.(*User)

	ac := &AccessCheck{}
	err := ak.UnmarshalPostAbort(ac)
	if err != nil {
		acr.Message = "Failed to parse access check object."
		ak.SetPayloadType("AccessCheckResult")
		ak.SetPayload(acr)
		ak.GinErrorAbort(401, "E401", "UnauthorizedAccess")
		return
	}

	acr.AccessChecked = ac

	_, checkAdmin := c.Get("AdminCheck")

	if (!checkAdmin && user.HasAccess(ac)) || (checkAdmin && user.HasAdminAccess(ac)) {
		acr.Status = true
		acr.Message = "Has access."
		ak.SetPayloadType("AccessCheckResult")
		ak.GinSend(acr)
		return
	}

	acr.Status = false
	acr.Message = "User does not have basic access."

	if checkAdmin {
		acr.Message = "User does not have admin access."
	}

	ak.SetPayload(acr)
	ak.GinErrorAbort(401, "E401", "UnauthorizedAccess")
}

// BasicAccess returns true is user is active and not locked
func (u *User) HasBasicAccess() bool {
	return u.Active
}

// BasicAccess returns true is user is active and not locked
func (u *User) HasAccess(ac *AccessCheck) bool {

	// is the user active
	//
	if !u.HasBasicAccess() {
		return false
	}

	// if user HasBasicAccess and is a SysOp then they
	// have access
	if u.Sysop {
		return true
	}

	// Admin check - if the user is an admin in the account
	if u.HasAdminAccess(ac) {
		return true
	}

	// At this point the user is valid but not a Sysop or
	// an admin of any of the accounts in the AccessCheck account
	// array.

	// Check for basic account access for ALL accounts listed in
	// the AccessCheck account array.
	for _, acc := range ac.Accounts {
		if !stringInSlice(acc, u.Accounts) {
			return false
		}
	}

	// Sections check
	// does the user have access to all sections?
	if !u.SectionsAll {
		// does config contain SECTIONS we need to check? and...
		// return false if ac.Sections has a section not in u.Sections
		for _, sec := range ac.Sections {
			if !stringInSlice(sec, u.Sections) {
				return false
			}
		}

	}

	return true
}

// HasAdminAccess
func (u *User) HasAdminAccess(ac *AccessCheck) bool {
	// is the user active
	//
	if !u.HasBasicAccess() {
		return false
	}

	// if user HasBasicAccess and is a SysOp then they
	// have access
	if u.Sysop {
		return true
	}

	// At this point we are a valid user but not a Sysop
	// so we need to ensure at least one account was specified.
	// You can not be an admin of nothing.
	if len(ac.Accounts) > 1 {
		return false
	}

	// Admin check - if the user is not an admin in ALL
	// of the accounts in the AccessCheck account array then
	// deny them
	for _, acc := range ac.Accounts {
		if !stringInSlice(acc, u.AdminAccounts) {
			return false
		}
	}

	// we must be an admin in all of the accounts provided
	return true
}

// stringInSlice util
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
