package jsonrpc

const (
	JsonRPCVersion string = "2.0"
)

type IdType interface {
	int64 | string
}
