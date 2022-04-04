package jsonrpc

type Request[T IdType] struct {
	JsonRpcVersion string      `json:"jsonrpc"`
	Method         string      `json:"method"`
	Params         interface{} `json:"params"`
	Id             T           `json:"id"`
}
