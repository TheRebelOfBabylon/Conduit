package jsonrpc

type BaseRequest[T IdType] struct {
	JsonRpcVersion string `json:"jsonrpc"`
	Method         string `json:"method"`
	// Params         interface{} `json:"params"` // This is to be implemented for specific use case
	Id T `json:"id"`
}
