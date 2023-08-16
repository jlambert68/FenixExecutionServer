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
func (executionEngine *TestInstructionExecutionEngineStruct) prepareTestInstructionIsNotHandledByThisExecutionInstanceSaveFinalTestInstructionExecutionResultToCloudDB(
	executionTrackNumber int,
	finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) {

	// Calculate Execution Track
	var executionTrack int
	executionTrack = common_config.CalculateExecutionTrackNumber(
		finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid)

	common_config.Logger.WithFields(logrus.Fields{
		"id": "34a32e60-6aa8-4400-ba2c-ff07a8f830e3",
		"finalTestInstructionExecutionResultMessage": finalTestInstructionExecutionResultMessage,
		"executionTrack": executionTrack,
	}).Debug("Incoming 'prepareTestInstructionIsNotHandledByThisExecutionInstanceSaveFinalTestInstructionExecutionResultToCloudDB'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "5fc72e58-f1fd-4c64-ab9d-28166bf02743",
	}).Debug("Outgoing 'prepareTestInstructionIsNotHandledByThisExecutionInstanceSaveFinalTestInstructionExecutionResultToCloudDB'")

	// Begin SQL Transaction
	var txn pgx.Tx
	var err error
	txn, err = fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"id":    "67830336-4592-40bf-b86e-e9f1720ebd85",
			"error": err,
			"finalTestInstructionExecutionResultMessage": finalTestInstructionExecutionResultMessage,
		}).Error("Problem to do 'DbPool.Begin'  in 'prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB', dropping 'finalTestInstructionExecutionResultMessage'")

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
	tempJson = protojson.Format(finalTestInstructionExecutionResultMessage)

	// Prepare message to be saved in database
	var tempTestInstructionExecutionMessageReceivedByWrongExecution common_config.TestInstructionExecutionMessageReceivedByWrongExecutionStruct
	tempTestInstructionExecutionMessageReceivedByWrongExecution = common_config.TestInstructionExecutionMessageReceivedByWrongExecutionStruct{
		ApplicatonExecutionRuntimeUuid:  common_config.ApplicationRuntimeUuid,
		TestInstructionExecutionUuid:    finalTestInstructionExecutionResultMessage.GetTestInstructionExecutionUuid(),
		TestInstructionExecutionVersion: 1,
		MessageType:                     common_config.FinalTestInstructionExecutionResultMessageType,
		MessageAsJsonString:             tempJson,
	}

	// Extract TestCaseExecutionQueue-messages to be added to data for ongoing Executions
	err = executionEngine.updateTestInstructionIsNotHandledByThisExecutionInstanceSaveToCloudDBInCloudDB(
		txn,
		tempTestInstructionExecutionMessageReceivedByWrongExecution)
	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "ee92c4fa-999a-47a8-aa64-00a6e00212c9",
			"error": err,
			"finalTestInstructionExecutionResultMessage": finalTestInstructionExecutionResultMessage,
		}).Error("Problem when saving 'finalTestInstructionExecutionResultMessage' to database in 'prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB', will dropp 'finalTestInstructionExecutionResultMessage'")

		return
	}

	// Create the BroadCastMessage for the TestInstructionExecution
	var testInstructionExecutionBroadcastMessages []common_config.
		TestInstructionExecutionBroadcastMessageStruct
	var testInstructionExecutionBroadcastMessage common_config.
		TestInstructionExecutionBroadcastMessageStruct

	testInstructionExecutionBroadcastMessage = common_config.
		TestInstructionExecutionBroadcastMessageStruct{
		TestInstructionExecutionUuid:                                finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid,
		TestInstructionExecutionVersion:                             "1",
		TestInstructionExecutionMessageReceivedByWrongExecutionType: common_config.FinalTestInstructionExecutionResultMessageType,
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
