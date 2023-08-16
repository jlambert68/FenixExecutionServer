package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"context"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"strings"
	"time"
)

// Prepare for Saving the ongoing Execution of a new TestCaseExecutionUuid in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) prepareTestInstructionIsNotHandledByThisExecutionInstanceSaveTestInstructionExecutionResponseStatusToCloudDB(
	executionTrackNumber int,
	processTestInstructionExecutionResponseStatus *fenixExecutionServerGrpcApi.ProcessTestInstructionExecutionResponseStatus) {

	// Calculate Execution Track
	var executionTrack int
	executionTrack = common_config.CalculateExecutionTrackNumber(
		processTestInstructionExecutionResponseStatus.TestInstructionExecutionUuid)

	common_config.Logger.WithFields(logrus.Fields{
		"id": "8740ed93-03eb-4822-9dff-b3f757615b1f",
		"processTestInstructionExecutionResponseStatus": processTestInstructionExecutionResponseStatus,
		"executionTrack": executionTrack,
	}).Debug("Incoming 'prepareTestInstructionIsNotHandledByThisExecutionInstanceSaveTestInstructionExecutionResponseStatusToCloudDB'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "ee9fee1c-181e-44c1-a36f-5a2f8f81ce85",
	}).Debug("Outgoing 'prepareTestInstructionIsNotHandledByThisExecutionInstanceSaveTestInstructionExecutionResponseStatusToCloudDB'")

	// Begin SQL Transaction
	var txn pgx.Tx
	var err error
	txn, err = fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"id":    "e2181314-3e10-41a8-9bf1-0ce7c5891248",
			"error": err,
			"processTestInstructionExecutionResponseStatus": processTestInstructionExecutionResponseStatus,
		}).Error("Problem to do 'DbPool.Begin'  in 'prepareTestInstructionIsNotHandledByThisExecutionInstanceSaveTestInstructionExecutionResponseStatusToCloudDB', dropping 'finalTestInstructionExecutionResultMessage'")

		return
	}

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Message to be sent over Broadcast-system that this ExecutionInstance is not responsible for this TestInstructionExecution
	var broadcastingMessageForExecutions common_config.
		BroadcastingMessageForTestInstructionExecutionsStruct

	// Standard is to do a Rollback
	doCommitNotRoleBack = false

	defer executionEngine.commitOrRoleBackTestInstructionIsNotHandledByThisExecutionInstance(
		&txn,
		&doCommitNotRoleBack,
		&broadcastingMessageForExecutions)

	// Generate json from on 'finalTestInstructionExecutionResultMessage'
	var tempJson string
	tempJson = protojson.Format(processTestInstructionExecutionResponseStatus)

	// Prepare message to be saved in database
	var tempTestInstructionExecutionMessageReceivedByWrongExecution common_config.TestInstructionExecutionMessageReceivedByWrongExecutionStruct
	tempTestInstructionExecutionMessageReceivedByWrongExecution = common_config.TestInstructionExecutionMessageReceivedByWrongExecutionStruct{
		ApplicatonExecutionRuntimeUuid:  common_config.ApplicationRuntimeUuid,
		TestInstructionExecutionUuid:    processTestInstructionExecutionResponseStatus.GetTestInstructionExecutionUuid(),
		TestInstructionExecutionVersion: 1,
		MessageType:                     common_config.ProcessTestInstructionExecutionResponseStatus,
		MessageAsJsonString:             tempJson,
	}

	// Extract TestCaseExecutionQueue-messages to be added to data for ongoing Executions
	err = executionEngine.updateTestInstructionIsNotHandledByThisExecutionInstanceSaveToCloudDBInCloudDB(
		txn,
		tempTestInstructionExecutionMessageReceivedByWrongExecution)
	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "c34396c6-ed28-46de-97a2-d92a62350c2b",
			"error": err,
			"processTestInstructionExecutionResponseStatus": processTestInstructionExecutionResponseStatus,
		}).Error("Problem when saving 'processTestInstructionExecutionResponseStatus' to database in 'prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB', will dropp 'finalTestInstructionExecutionResultMessage'")

		return
	}

	// Create the BroadCastMessage for the TestInstructionExecution
	var testInstructionExecutionBroadcastMessages []common_config.
		TestInstructionExecutionBroadcastMessageStruct
	var testInstructionExecutionBroadcastMessage common_config.
		TestInstructionExecutionBroadcastMessageStruct

	testInstructionExecutionBroadcastMessage = common_config.
		TestInstructionExecutionBroadcastMessageStruct{
		TestInstructionExecutionUuid:                                processTestInstructionExecutionResponseStatus.TestInstructionExecutionUuid,
		TestInstructionExecutionVersion:                             "1",
		TestInstructionExecutionMessageReceivedByWrongExecutionType: common_config.ProcessTestInstructionExecutionResponseStatus,
	}
	testInstructionExecutionBroadcastMessages = append(testInstructionExecutionBroadcastMessages,
		testInstructionExecutionBroadcastMessage)

	broadcastingMessageForExecutions = common_config.
		BroadcastingMessageForTestInstructionExecutionsStruct{
		OriginalMessageCreationTimeStamp: strings.Split(time.Now().UTC().String(), " m=")[0],
		TestInstructionExecutions:        testInstructionExecutionBroadcastMessages,
	}

	// Do the commit and send over Broadcast-system
	doCommitNotRoleBack = true
}
