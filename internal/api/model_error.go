package api

// Error struct for Error
type Error struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}
