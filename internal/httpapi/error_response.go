package httpapi

// errorResponse is the stable JSON shape returned for every API error.
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// errorCode maps HTTP status codes to stable machine-readable identifiers.
func errorCode(status int) string {
	switch status {
	case 400:
		return "bad_request"
	case 401:
		return "unauthorized"
	case 403:
		return "forbidden"
	case 404:
		return "not_found"
	case 405:
		return "method_not_allowed"
	case 409:
		return "conflict"
	case 413:
		return "payload_too_large"
	case 416:
		return "range_not_satisfiable"
	case 503:
		return "service_unavailable"
	default:
		return "internal_error"
	}
}

