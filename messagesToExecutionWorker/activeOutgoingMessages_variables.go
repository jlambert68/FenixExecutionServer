package messagesToExecutionWorker

import (
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
)

type MessagesToExecutionWorkerServerObjectStruct struct {
	logger         *logrus.Logger
	gcpAccessToken *oauth2.Token
}

// Variables used for contacting Fenix Execution Worker Server
var (
	remoteFenixExecutionWorkerServerConnection *grpc.ClientConn
	FenixExecutionServerAddressToDial          string
	fenixExecutionWorkerServerGrpcClient       fenixExecutionServerGrpcApi.FenixExecutionServerGrpcServicesClient
)
