package parentproxy

const (
	PROXY_TYPE_RABBITMQ = "rabbitmq"
	PROXY_TYPE_KMS      = "kms"
	PROXY_READY         = "proxy_ready"
)

type Message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type Proxy interface {
	Handle(data map[string]interface{}) interface{}
}
