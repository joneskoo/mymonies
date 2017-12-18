package api

var (
	badParametersError  = errorResponse{4001, "parameters failed input validation"}
	internalServerError = errorResponse{5000, "internal server error"}
	jsonEncodingError   = errorResponse{5001, "failed to encode JSON"}
)

type errorResponse struct {
	// Code is the logical error code.
	Code int

	// Message is the error message returned to caller.
	// The message must not contain sensitive details.
	Message string
}

func (e errorResponse) Error() string {
	return e.Message
}

// HTTPCode is the HTTP status code of the error.
func (e errorResponse) HTTPCode() int {
	return e.Code / 10
}
