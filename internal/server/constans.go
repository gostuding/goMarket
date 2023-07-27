package server

const (
	writeResponceErrorString = "responce body write error: %v"
	iconPath                 = "/favicon.ico"
	authHeader               = "Authorization"
	contentTypeString        = "Content-Type"
	ctApplicationJSONString  = "application/json"
	checkOrderErrorString    = "check order error"
	bodyReadError            = "orders body read error"
	jsonConvertEerrorString  = "convert to json error"
	uidConvertErrorString    = "uid convert error"
	uidContextTypeError      = "context uid is not int"
	incorrectIPErroString    = "remote ip incorrect: %w"
	validateError            = "request validate error: %w"
	gormError                = "gorm error: %w"
	tokenGenerateError       = "token generation error: %w"
	readRequestErrorString   = "read request body error: %w"
)
