package main

import (
	"FenixExecutionServer/common_config"
	"context"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"github.com/sirupsen/logrus"
)

// ReportProcessingCapability - *********************************************************************
// Client can inform Server of Client capability to execute requests in parallell, serial or no processing at all
func (s *fenixExecutionServerGrpcServicesServer) ReportProcessingCapability(ctx context.Context, processingCapabilityMessage *fenixExecutionServerGrpcApi.ProcessingCapabilityMessage) (*fenixExecutionServerGrpcApi.AckNackResponse, error) {

	fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "fd6f4e08-7708-454c-932c-231269628031",
	}).Debug("Incoming 'gRPC - ReportProcessingCapability'")

	defer fenixExecutionServerObject.logger.WithFields(logrus.Fields{
		"id": "6b75f162-17c5-4c6c-a31c-502a9e76a826",
	}).Debug("Outgoing 'gRPC - ReportProcessingCapability'")

	// Current user
	userID := processingCapabilityMessage.ClientSystemIdentification.DomainUuid

	// Check if Client is using correct proto files version
	returnMessage := common_config.IsClientUsingCorrectTestDataProtoFileVersion(userID, fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(processingCapabilityMessage.ClientSystemIdentification.ProtoFileVersionUsedByClient))
	if returnMessage != nil {

		// Exiting
		return returnMessage, nil
	}

	return &fenixExecutionServerGrpcApi.AckNackResponse{AckNack: true, Comments: ""}, nil
}
