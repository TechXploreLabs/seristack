package errors

type ErrorCode string

const (
	METHOD_NOT_ALLOWED ErrorCode = "METHOD_NOT_ALLOWED"
	BAD_REQUEST        ErrorCode = "BAD_REQUEST"
	INTERNAL_ERROR     ErrorCode = "INTERNAL_ERROR"
	NOT_FOUND          ErrorCode = "NOT_FOUND"
)

func (e ErrorCode) String() string {
	return string(e)
}
