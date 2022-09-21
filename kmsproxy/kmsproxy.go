package kmsproxy

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

const (
	OPTERATION_ENCRYPT = "encrypt"
)

type KmsProxy struct {
	logger    *log.Logger
	Conn      net.Conn
	AWSRegion string
	KeyId     string
}

func New(conn net.Conn, region string, keyId string) *KmsProxy {
	return &KmsProxy{
		logger:    log.New(os.Stdout, "[kmsproxy] ", log.Ldate|log.Ltime|log.Lshortfile),
		Conn:      conn,
		AWSRegion: region,
		KeyId:     keyId,
	}
}

func (proxy *KmsProxy) Handle(data map[string]interface{}) interface{} {
	if op, ok := data["op"]; ok {
		switch op {
		case OPTERATION_ENCRYPT:
			if _d, ok := data["data"].(string); ok {
				proxy.logger.Printf("data is %v", _d)
				cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(proxy.AWSRegion))
				if err != nil {
					log.Fatalf("Unable to load AWS SDK config, %v", err)
				}
				kmsSDK := kms.NewFromConfig(cfg)
				params := kms.EncryptInput{
					KeyId:     &proxy.KeyId,
					Plaintext: []byte(_d),
				}
				if out, err := kmsSDK.Encrypt(context.Background(), &params); err != nil {
					proxy.logger.Printf("Encrypt failed. %v", err)
					return nil
				} else {
					proxy.logger.Printf("Encrypt cipher is: %s", string(out.CiphertextBlob))
					return _d
				}
			} else {
				proxy.logger.Printf("data is not a string. %v", data["data"])
				return nil
			}
		default:
			proxy.logger.Printf("No operation match. %s", op)
			return nil
		}
	} else {
		proxy.logger.Printf("op is not a string. %v", data["op"])
		return nil
	}
}
