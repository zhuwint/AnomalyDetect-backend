package dapr

import (
	"anomaly-detect/pkg/env"
	"log"
)

const (
	GrpcProtocol       = "grpc"
	HttpProtocol       = "http"
	AppIdEnv           = "APP_ID"
	AppPortEnv         = "APP_PORT"
	AppApiTokenEnv     = "APP_API_TOKEN"
	daprHttpPortEnv    = "DAPR_HTTP_PORT"
	daprGrpcPortEnv    = "DAPR_GRPC_PORT"
	daprMetricsPortEnv = "DAPR_METRICS_PORT"
)

const (
	//invokeUrl        = "http://localhost:%d/v1.0/invoke/%s/method/%s"
	//getStateUrl      = "http://localhost:%d/v1.0/state/%s/%s"
	//saveStateUrl     = "http://localhost:%d/v1.0/state/%s"
	//outputBindingUrl = "http://localhost:%d/v1.0/bindings/%s"
	//publishUrl       = "http://localhost:%d/v1.0/publish/%s/%s"
	healthzUrl = "http://localhost:%d/v1.0/healthz"
)

type Dapr struct {
	AppName      string
	AppPort      int
	DaprHttpPort int
	DaprGrpcPort int
}

func NewDapr(service string, port int) *Dapr {
	dapr := &Dapr{
		AppName: service,
		AppPort: port,
	}
	dapr.DaprHttpPort = env.GetEnvInt(daprHttpPortEnv, 3500)
	dapr.DaprGrpcPort = env.GetEnvInt(daprGrpcPortEnv, 4500)
	log.Printf("app-id=%s app-port=%d dapr-http-port=%d dapr-grpc-port=%d",
		service, port, dapr.DaprHttpPort, dapr.DaprGrpcPort)
	return dapr
}
