package jsonrpc

type ResponseErrorCode int32

const (
	JSONRPC_PARSE_ERR            ResponseErrorCode = -32700
	JSONRPC_INVALID_REQUEST_ERR  ResponseErrorCode = -32600
	JSONRPC_METHOD_NOT_FOUND_ERR ResponseErrorCode = -32601
	JSONRPC_INVALID_PARAMS_ERR   ResponseErrorCode = -32602
	JSONRPC_INTERNAL_ERR         ResponseErrorCode = -32603
	JSONRPC_SERVER_ERR_MIN       ResponseErrorCode = -32000
	JSONRPC_SERVER_ERR_MAX       ResponseErrorCode = -32099
)

type BaseResponseError struct {
	Code    ResponseErrorCode `json:"code"`
	Message string            `json:"message"`
	// Data    interface{}       `json:"data"` // This is to be implemented for specific use cases
}

type BaseResponse[T IdType] struct {
	JsonRpcVersion string `json:"jsonrpc"`
	// Result         interface{}   `json:"result"` // This is to be implemented for specific use cases
	Err BaseResponseError `json:"error"`
	Id  T                 `json:"id"`
}
