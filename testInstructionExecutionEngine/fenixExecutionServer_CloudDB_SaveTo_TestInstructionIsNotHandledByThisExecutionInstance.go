package testInstructionExecutionEngine

import (
	broadcastingEngine_TestInstructionNotHandledByThisInstance "FenixExecutionServer/broadcastEngine_TestInstructionNotHandledByThisInstance"
	"FenixExecutionServer/common_config"
	"context"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

// Commit or Rollback changes 'TestInstructionIsNotHandledByThisExecutionInstanceUpSaveToCloudDB' and send over Broadcast-channel
func (executionEngine *TestInstructionExecutionEngineStruct) commitOrRoleBackTestInstructionIsNotHandledByThisExecutionInstance(
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool,
	broadcastingMessageForExecutionsReference *broadcastingEngine_TestInstructionNotHandledByThisInstance.BroadcastingMessageForExecutionsStruct) {

	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference
	broadcastingMessageForExecutions := *broadcastingMessageForExecutionsReference

	// Should transaction be committed and be broadcast
	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

		// Send message to BroadcastEngine over channel
		broadcastingEngine_TestInstructionNotHandledByThisInstance.BroadcastEngineMessageChannel <- broadcastingMessageForExecutions

		common_config.Logger.WithFields(logrus.Fields{
			"id":                               "f88f7282-be22-43eb-bed0-a6e600d3db99",
			"broadcastingMessageForExecutions": broadcastingMessageForExecutions,
		}).Debug("Sent message for broadcasting (broadcastingEngine_TestInstructionNotHandledByThisInstance)")

	} else {

		dbTransaction.Rollback(context.Background())
	}
}

// Prepare for Saving the ongoing Execution of a new TestCaseExecutionUuid in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) prepareTestInstructionIsNotHandledByThisExecutionInstanceUpSaveToCloudDB(
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
	}).Debug("Incoming 'prepareTestInstructionIsNotHandledByThisExecutionInstanceUpSaveToCloudDB'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "5fc72e58-f1fd-4c64-ab9d-28166bf02743",
	}).Debug("Outgoing 'prepareTestInstructionIsNotHandledByThisExecutionInstanceUpSaveToCloudDB'")

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
	var broadcastingMessageForExecutions broadcastingEngine_TestInstructionNotHandledByThisInstance.
		BroadcastingMessageForExecutionsStruct

	// Standard is to do a Rollback
	doCommitNotRoleBack = false

	defer executionEngine.commitOrRoleBackTestInstructionIsNotHandledByThisExecutionInstance(
		&txn,
		&doCommitNotRoleBack,
		&broadcastingMessageForExecutions)

	// Extract TestCaseExecutionQueue-messages to be added to data for ongoing Executions
	err = executionEngine.updateTestInstructionIsNotHandledByThisExecutionInstanceSaveToCloudDBInCloudDB2(txn, finalTestInstructionExecutionResultMessage)
	if err != nil {

		common_config.Logger.WithFields(logrus.Fields{
			"id":    "ee92c4fa-999a-47a8-aa64-00a6e00212c9",
			"error": err,
			"finalTestInstructionExecutionResultMessage": finalTestInstructionExecutionResultMessage,
		}).Error("Problem to do 'DbPool.Begin'  in 'prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB', dropping 'finalTestInstructionExecutionResultMessage'")

		return
	}

	// Create the BroadCastMessage for the TestInstructionExecution
	var testInstructionExecutionBroadcastMessages []broadcastingEngine_TestInstructionNotHandledByThisInstance.
		TestInstructionExecutionBroadcastMessageStruct
	var testInstructionExecutionBroadcastMessage broadcastingEngine_TestInstructionNotHandledByThisInstance.
		TestInstructionExecutionBroadcastMessageStruct

	testInstructionExecutionBroadcastMessage = broadcastingEngine_TestInstructionNotHandledByThisInstance.
		TestInstructionExecutionBroadcastMessageStruct{
		TestInstructionExecutionUuid:    finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid,
		TestInstructionExecutionVersion: "1",
	}
	testInstructionExecutionBroadcastMessages = append(testInstructionExecutionBroadcastMessages,
		testInstructionExecutionBroadcastMessage)

	broadcastingMessageForExecutions = broadcastingEngine_TestInstructionNotHandledByThisInstance.
		BroadcastingMessageForExecutionsStruct{
		OriginalMessageCreationTimeStamp: strings.Split(time.Now().UTC().String(), " m=")[0],
		TestInstructionExecutions:        testInstructionExecutionBroadcastMessages,
	}

	// Do the commit and send over Broadcast-system
	doCommitNotRoleBack = true
}
