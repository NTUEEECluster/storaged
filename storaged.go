package storaged

import "encoding/json"

type RequestType string

const (
	RequestCheckQuota RequestType = "check_quota"
	RequestUpdate     RequestType = "update"
	RequestDelete     RequestType = "delete"
)

type Request struct {
	Type    RequestType     `json:"type"`
	Content json.RawMessage `json:"content,omitempty"`
}

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

type DeleteRequest struct {
	// Name is the name of the folder to delete.
	Name string `json:"name"`
}
