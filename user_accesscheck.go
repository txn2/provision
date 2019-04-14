package provision

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
