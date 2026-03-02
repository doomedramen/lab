package model

// ISOImage represents an ISO image file
type ISOImage struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Size        int64   `json:"size"`     // in bytes
	Path        string  `json:"path"`     // full path to file
	Description string  `json:"description"`
	OS          string  `json:"os"`       // detected OS type
	Status      string  `json:"status"`   // available, uploading, error
	CreatedAt   string  `json:"createdAt"`
}

// ISOListResponse represents the response for listing ISOs
type ISOListResponse struct {
	ISOs  []ISOImage `json:"isos"`
	Total int        `json:"total"`
}
