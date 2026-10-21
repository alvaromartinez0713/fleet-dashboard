package domain

// SaveOutcome represents the result of a Save operation.
// It distinguishes between successful new insertion and successful deduplication.
// Both cases represent successful persistence - the event exists in the database.
//
// This type is part of the domain because persistence outcome is a domain concern:
// the use case needs to know whether this is a new event or a retry/duplicate.
type SaveOutcome struct {
	// IsNew indicates whether this was a new insertion (true) or a duplicate (false).
	// Only meaningful when Error is nil.
	IsNew bool

	// Error contains any error that occurred during persistence.
	// If Error is not nil, IsNew should be ignored.
	Error error
}

// Success returns true if persistence succeeded (no error).
func (so *SaveOutcome) Success() bool {
	return so.Error == nil
}

// WasInserted returns true if this was a new insertion (not a duplicate).
// Only valid if Success() is true.
func (so *SaveOutcome) WasInserted() bool {
	return so.Success() && so.IsNew
}

// WasDuplicate returns true if this was a duplicate (already existed, ON CONFLICT triggered).
// Only valid if Success() is true.
func (so *SaveOutcome) WasDuplicate() bool {
	return so.Success() && !so.IsNew
}
