package storaged

type CheckQuotaRequest struct {
	User string `json:"user,omitempty"`
}

type UpdateRequest struct {
	// Name is the name of the folder to create.
	Name string `json:"name"`
	// Tier is the storage tier to request for, i.e. "ssd" or "hdd".
	Tier string `json:"tier"`
	// SizeInGB is the quota to assign to the folder.
	SizeInGB int `json:"size_in_gb"`
}
