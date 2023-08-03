package server

const (
	defaultAuthTokenLiveTime      = 3600
	defaultAccrualRequestInterval = 1
	manyRequestsWaitTimeDef       = 60
	shutdownTimeout               = 10
	writeResponceErrorString      = "responce body write error: %w"
	contentTypeString             = "Content-Type"
	ctApplicationJSONString       = "application/json"
	uidContextTypeError           = "context uid is not int"
	incorrectIPErroString         = "remote ip incorrect: %w"
	gormError                     = "gorm error: %w"
	tokenGenerateError            = "token generation error: %w"
	readRequestErrorString        = "read request body error: %w"
)
