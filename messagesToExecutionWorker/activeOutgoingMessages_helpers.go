package messagesToExecutionWorker

import (
	"FenixExecutionServer/common_config"
	"crypto/tls"
	"fmt"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/idtoken"
	grpcMetadata "google.golang.org/grpc/metadata"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"golang.org/x/net/context"
)

// ********************************************************************************************************************

// Extract reference to WorkerVariablesStruct

func (fenixExecutionWorkerObject *MessagesToExecutionWorkerServerObjectStruct) getWorkerVariablesReference(domainUuid string) (executionWorkerVariablesReference *common_config.ExecutionWorkerVariablesStruct) {

	executionWorkerVariablesReference, existInMap := common_config.ExecutionWorkerVariablesMap[domainUuid]
	if existInMap == false {
		fenixExecutionWorkerObject.Logger.WithFields(logrus.Fields{
			"ID":         "bb2d7e51-83dc-4027-9c6a-ee8b4677699f",
			"domainUuid": domainUuid,
		}).Fatalln(fmt.Sprintf("Couldn't find DomainUuid: %s. This shouldn't happend", domainUuid))
	}

	return executionWorkerVariablesReference
}

// SetConnectionToFenixTestExecutionServer - Set upp connection and Dial to FenixExecutionServer
func (fenixExecutionWorkerObject *MessagesToExecutionWorkerServerObjectStruct) SetConnectionToFenixTestExecutionServer(domainUuid string) (err error) {

	// Get WorkerVariablesReference
	workerVariables := fenixExecutionWorkerObject.getWorkerVariablesReference(domainUuid)

	var opts []grpc.DialOption

	//When running on GCP then use credential otherwise not
	if common_config.ExecutionLocationForWorker == common_config.GCP {
		creds := credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})

		opts = []grpc.DialOption{
			grpc.WithTransportCredentials(creds),
		}
	}

	// Set up connection to Fenix Worker Server
	// When run on GCP, use credentials
	if common_config.ExecutionLocationForWorker == common_config.GCP {
		// Run on GCP
		workerVariables.RemoteFenixExecutionWorkerServerConnection, err = grpc.Dial(workerVariables.FenixExecutionServerAddressToDial, opts...)
	} else {
		// Run Local
		workerVariables.RemoteFenixExecutionWorkerServerConnection, err = grpc.Dial(workerVariables.FenixExecutionServerAddressToDial, grpc.WithInsecure())
	}
	if err != nil {
		fenixExecutionWorkerObject.Logger.WithFields(logrus.Fields{
			"ID": "50b59b1b-57ce-4c27-aa84-617f0cde3100",
			"workerVariables.FenixExecutionServerAddressToDial": workerVariables.FenixExecutionServerAddressToDial,
			"error message": err,
			"domainUuid":    domainUuid,
		}).Error("Did not connect to FenixExecutionWorkerServer via gRPC")

		return err

	} else {
		fenixExecutionWorkerObject.Logger.WithFields(logrus.Fields{
			"ID": "0c650bbc-45d0-4029-bd25-4ced9925a059",
			"workerVariables.FenixExecutionServerAddressToDial": workerVariables.FenixExecutionServerAddressToDial,
			"domainUuid": domainUuid,
		}).Info("gRPC connection OK to FenixExecutionWorkerServer")

		// Creates a new Clients
		workerVariables.FenixExecutionWorkerServerGrpcClient = fenixExecutionServerGrpcApi.NewFenixExecutionServerGrpcServicesClient(workerVariables.RemoteFenixExecutionWorkerServerConnection)

	}
	return err
}

// Generate Google access token. Used when running in GCP
func (fenixExecutionWorkerObject *MessagesToExecutionWorkerServerObjectStruct) generateGCPAccessToken(ctx context.Context, domainUuid string) (appendedCtx context.Context, returnAckNack bool, returnMessage string) {

	// Get WorkerVariablesReference
	workerVariables := fenixExecutionWorkerObject.getWorkerVariablesReference(domainUuid)

	// Only create the token if there is none, or it has expired
	if fenixExecutionWorkerObject.gcpAccessToken == nil || fenixExecutionWorkerObject.gcpAccessToken.Expiry.Before(time.Now()) {

		// Create an identity token.
		// With a global TokenSource tokens would be reused and auto-refreshed at need.
		// A given TokenSource is specific to the audience.
		tokenSource, err := idtoken.NewTokenSource(ctx, "https://"+workerVariables.FenixExecutionServerWorkerAddressToUse)
		if err != nil {
			fenixExecutionWorkerObject.Logger.WithFields(logrus.Fields{
				"ID":  "8ba622d8-b4cd-46c7-9f81-d9ade2568eca",
				"err": err,
			}).Error("Couldn't generate access token")

			return nil, false, "Couldn't generate access token"
		}

		token, err := tokenSource.Token()
		if err != nil {
			fenixExecutionWorkerObject.Logger.WithFields(logrus.Fields{
				"ID":  "0cf31da5-9e6b-41bc-96f1-6b78fb446194",
				"err": err,
			}).Error("Problem getting the token")

			return nil, false, "Problem getting the token"
		} else {
			fenixExecutionWorkerObject.Logger.WithFields(logrus.Fields{
				"ID":    "8b1ca089-0797-4ee6-bf9d-f9b06f606ae9",
				"token": token,
			}).Debug("Got Bearer Token")
		}

		fenixExecutionWorkerObject.gcpAccessToken = token

	}

	fenixExecutionWorkerObject.Logger.WithFields(logrus.Fields{
		"ID": "cd124ca3-87bb-431b-9e7f-e044c52b4960",
		"FenixExecutionWorkerObject.gcpAccessToken": fenixExecutionWorkerObject.gcpAccessToken,
	}).Debug("Will use Bearer Token")

	// Add token to GrpcServer Request.
	appendedCtx = grpcMetadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+fenixExecutionWorkerObject.gcpAccessToken.AccessToken)

	return appendedCtx, true, ""

}
