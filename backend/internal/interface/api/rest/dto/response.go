package dto

//easyjson:json
type BBoxDTO struct {
	XMin float64 `json:"xMin"`
	YMin float64 `json:"yMin"`
	XMax float64 `json:"xMax"`
	YMax float64 `json:"yMax"`
}

//easyjson:json
type PredictionDTO struct {
	ClassID    string  `json:"classId"`
	Confidence float64 `json:"confidence"`
	BBox       BBoxDTO `json:"bbox"`
}

//easyjson:json
type PhotoDTO struct {
	ID          string          `json:"id"`
	X           float64         `json:"x"`
	Y           float64         `json:"y"`
	H           float64         `json:"h"`
	Width       int             `json:"width"`
	Height      int             `json:"height"`
	CapturedAt  string          `json:"capturedAt"`
	OriginalURL string          `json:"originalUrl"`
	Predictions []PredictionDTO `json:"predictions"`
}

//easyjson:json
type PhotoPageDTO struct {
	Items     []PhotoDTO `json:"items"`
	NextToken string     `json:"next_token,omitempty"`
}

//easyjson:json
type UploadLinkDTO struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt string            `json:"expiresAt"`
	Exists    bool              `json:"exists"`
}

//easyjson:json
type LoginResponseDTO struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

//easyjson:json
type FieldErrorDTO struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

//easyjson:json
type ErrorDTO struct {
	RequestID  string          `json:"request_id"`
	Message    string          `json:"message"`
	Code       string          `json:"code,omitempty"`
	ResourceID string          `json:"resource_id,omitempty"`
	Details    []FieldErrorDTO `json:"details,omitempty"`
}
