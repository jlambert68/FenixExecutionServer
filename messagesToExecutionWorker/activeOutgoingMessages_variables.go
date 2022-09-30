package messagesToExecutionWorker

import (
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type MessagesToExecutionWorkerServerObjectStruct struct {
	Logger         *logrus.Logger
	gcpAccessToken *oauth2.Token
}

// Variables used for contacting Fenix Execution Worker Server
/*
var (
	remoteFenixExecutionWorkerServerConnection *grpc.ClientConn
	FenixExecutionServerAddressToDial          string
	fenixExecutionWorkerServerGrpcClient       fenixExecutionServerGrpcApi.FenixExecutionServerGrpcServicesClient
)

*/
