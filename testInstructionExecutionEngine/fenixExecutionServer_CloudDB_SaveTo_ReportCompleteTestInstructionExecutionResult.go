package testInstructionExecutionEngine

import (
	"FenixExecutionServer/broadcastingEngine_ExecutionStatusUpdate"
	"FenixExecutionServer/common_config"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

func (executionEngine *TestInstructionExecutionEngineStruct) commitOrRoleBackReportCompleteTestInstructionExecutionResult(
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool,
	testCaseExecutionsToProcessReference *[]ChannelCommandTestCaseExecutionStruct,
	thereExistsOnGoingTestInstructionExecutionsReference *bool,
	triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutionsReference *bool,
	testInstructionExecutionReference *broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct) {

	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference
	testCaseExecutionsToProcess := *testCaseExecutionsToProcessReference
	thereExistsOnGoingTestInstructionExecutions := *thereExistsOnGoingTestInstructionExecutionsReference
	triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutions := *triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutionsReference
	testInstructionExecution := *testInstructionExecutionReference

	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

		// Create message to be sent to BroadcastEngine
		var broadcastingMessageForExecutions broadcastingEngine_ExecutionStatusUpdate.BroadcastingMessageForExecutionsStruct
		broadcastingMessageForExecutions = broadcastingEngine_ExecutionStatusUpdate.BroadcastingMessageForExecutionsStruct{
			OriginalMessageCreationTimeStamp: strings.Split(time.Now().UTC().String(), " m=")[0],
			TestCaseExecutions:               nil,
			TestInstructionExecutions:        []broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct{testInstructionExecution},
		}

		// Send message to BroadcastEngine over channel
		broadcastingEngine_ExecutionStatusUpdate.BroadcastEngineMessageChannel <- broadcastingMessageForExecutions

		common_config.Logger.WithFields(logrus.Fields{
			"id":                               "c71501df-2ddc-4874-953f-4cca70d9b698",
			"broadcastingMessageForExecutions": broadcastingMessageForExecutions,
		}).Debug("Sent message on Broadcast channel")

		// Remove TestInstructionExecution from TimeOut-timer

		// Convert strings to integer
		var tempTestCaseExecutionVersion int
		var tempTestInstructionExecutionVersionAsInteger int
		var err error

		// TestCaseExecutionVersion
		tempTestCaseExecutionVersion, err = strconv.Atoi(testInstructionExecution.TestCaseExecutionVersion)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"id": "285c4321-fec5-402a-8874-50360695efcf",
				"testInstructionExecution.TestCaseExecutionVersion": testInstructionExecution.TestCaseExecutionVersion,
			}).Error("Couldn't convert string-version of 'TestCaseExecutionVersion' to an integer")

			return
		}

		// TestInstructionExecutionVersion
		tempTestInstructionExecutionVersionAsInteger, err = strconv.Atoi(testInstructionExecution.TestInstructionExecutionVersion)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"id": "499cc96e-cc02-4247-94b7-70f08beeaa20",
				"testInstructionExecution.TestInstructionExecutionVersion": testInstructionExecution.TestInstructionExecutionVersion,
			}).Error("Couldn't convert string-version of 'TestInstructionExecutionVersion' to an integer")

			return

		}

		// Create a message with TestInstructionExecution to be sent to TimeOutEngine
		var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
		tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
			TestCaseExecutionUuid:           testInstructionExecution.TestCaseExecutionUuid,
			TestCaseExecutionVersion:        int32(tempTestCaseExecutionVersion),
			TestInstructionExecutionUuid:    testInstructionExecution.TestInstructionExecutionUuid,
			TestInstructionExecutionVersion: int32(tempTestInstructionExecutionVersionAsInteger),
			//TestInstructionExecutionCanBeReExecuted: false,
			//TimeOutTime:                             nil,
		}

		var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
		tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
			TimeOutChannelCommand:                   common_config.TimeOutChannelCommandRemoveTestInstructionExecutionFromTimeOutTimerDueToExecutionResult,
			TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutions,
			//TimeOutReturnChannelForTimeOutHasOccurred:                           nil,
			//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer: nil,
			SendID:                         "18d960b0-a0dc-4058-9370-c66dce099e3d",
			MessageInitiatedFromPubSubSend: false,
		}

		// Calculate Execution Track
		var executionTrack int
		executionTrack = common_config.CalculateExecutionTrackNumber(
			testInstructionExecution.TestInstructionExecutionUuid)

		// Send message on TimeOutEngineChannel to Add TestInstructionExecution to Timer-queue
		*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrack] <- tempTimeOutChannelCommand

		// Update status for TestCaseExecutionUuid, based on incoming TestInstructionExecution
		if triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutions == true {

			// Create response channel to be able to get response when ChannelCommand has finished
			var returnChannelWithDBError ReturnChannelWithDBErrorType
			returnChannelWithDBError = make(chan ReturnChannelWithDBErrorStruct)

			channelCommandMessage := ChannelCommandStruct{
				ChannelCommand:                    ChannelCommandUpdateExecutionStatusOnTestCaseExecutionExecutions,
				ChannelCommandTestCaseExecutions:  testCaseExecutionsToProcess,
				ReturnChannelWithDBErrorReference: &returnChannelWithDBError,
			}

			*executionEngine.CommandChannelReferenceSlice[executionTrack] <- channelCommandMessage

			// Wait for errReturnMessage in return channel
			var returnChannelMessage ReturnChannelWithDBErrorStruct
			returnChannelMessage = <-returnChannelWithDBError

			//Check if there was an error in previous ChannelCommand, if so then exit
			if returnChannelMessage.Err != nil {
				return
			}

			// Trigger TestInstructionEngine to check if there are any TestInstructions on the ExecutionQueue, If we got an OK as respons from TestInstruction

			channelCommandMessage = ChannelCommandStruct{
				ChannelCommand:                   ChannelCommandCheckForTestInstructionExecutionsWaitingToBeSentToWorker,
				ChannelCommandTestCaseExecutions: testCaseExecutionsToProcess,
			}

			*executionEngine.CommandChannelReferenceSlice[executionTrack] <- channelCommandMessage

		} else {

			// Create response channel to be able to get response when ChannelCommand has finished
			var returnChannelWithDBError ReturnChannelWithDBErrorType
			returnChannelWithDBError = make(chan ReturnChannelWithDBErrorStruct)

			// Update status for TestCaseExecutionUuid, based on incoming TestInstructionExecution
			channelCommandMessage := ChannelCommandStruct{
				ChannelCommand:                    ChannelCommandUpdateExecutionStatusOnTestCaseExecutionExecutions,
				ChannelCommandTestCaseExecutions:  testCaseExecutionsToProcess,
				ReturnChannelWithDBErrorReference: &returnChannelWithDBError,
			}

			*executionEngine.CommandChannelReferenceSlice[executionTrack] <- channelCommandMessage

			// Wait for errReturnMessage in return channel
			var returnChannelMessage ReturnChannelWithDBErrorStruct
			returnChannelMessage = <-returnChannelWithDBError

			//Check if there was an error in previous ChannelCommand, if so then exit
			if returnChannelMessage.Err != nil {
				return
			}

			// If there are Ongoing TestInstructionsExecutions then secure that they are triggered to be sent to Worker
			if thereExistsOnGoingTestInstructionExecutions == true {
				channelCommandMessage = ChannelCommandStruct{
					ChannelCommand:                   ChannelCommandCheckForTestInstructionExecutionsWaitingToBeSentToWorker, // ChannelCommandCheckOngoingTestInstructionExecutions,
					ChannelCommandTestCaseExecutions: testCaseExecutionsToProcess,
				}
				*executionEngine.CommandChannelReferenceSlice[executionTrack] <- channelCommandMessage

			}
		}
	} else {
		dbTransaction.Rollback(context.Background())
	}
}

// Prepare for Saving the ongoing Execution of a new TestCaseExecutionUuid in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB(
	executionTrackNumber int,
	finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (ackNackResponse *fenixExecutionServerGrpcApi.AckNackResponse) {

	// Verify that the ExecutionStatus is a final status
	// (0, 'TIE_INITIATED') -> NOT OK
	// (1, 'TIE_EXECUTING') -> NOT OK
	// (2, 'TIE_CONTROLLED_INTERRUPTION' -> OK
	// (3, 'TIE_CONTROLLED_INTERRUPTION_CAN_BE_RERUN' -> OK
	// (4, 'TIE_FINISHED_OK' -> OK
	// (5, 'TIE_FINISHED_OK_CAN_BE_RERUN' -> OK
	// (6, 'TIE_FINISHED_NOT_OK' -> OK
	// (7, 'TIE_FINISHED_NOT_OK_CAN_BE_RERUN' -> OK
	// (8, 'TIE_UNEXPECTED_INTERRUPTION' -> OK
	// (9, 'TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN' -> OK
	// (10, 'TIE_TIMEOUT_INTERRUPTION_CAN_BE_RERUN' -> OK
	// (11, 'TIE_TIMEOUT_INTERRUPTION' -> OK

	// Calculate Execution Track
	var executionTrack int
	executionTrack = common_config.CalculateExecutionTrackNumber(
		finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid)

	common_config.Logger.WithFields(logrus.Fields{
		"id": "51054338-e8ac-49ef-bb30-0efebc98e029",
		"finalTestInstructionExecutionResultMessage": finalTestInstructionExecutionResultMessage,
		"executionTrack": executionTrack,
	}).Debug("Incoming 'prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "4601d6d7-f7b7-428e-957b-eeec12debf74",
	}).Debug("Outgoing 'prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB'")

	// Create Response channel from TimeOutEngine to get answer if TestInstructionExecution Already has been TimedOut
	var timeOutResponseChannelForTimeOutHasOccurred common_config.TimeOutResponseChannelForTimeOutHasOccurredType
	timeOutResponseChannelForTimeOutHasOccurred = make(chan common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct)

	// Create a message with TestInstructionExecution to be sent to TimeOutEngine for check if it has TimedOut
	var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
	tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
		TestInstructionExecutionUuid:    finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid,
		TestInstructionExecutionVersion: 1,
	}

	var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
	tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
		TimeOutChannelCommand:                     common_config.TimeOutChannelCommandHasTestInstructionExecutionAlreadyTimedOut,
		TimeOutChannelTestInstructionExecutions:   tempTimeOutChannelTestInstructionExecutions,
		TimeOutReturnChannelForTimeOutHasOccurred: &timeOutResponseChannelForTimeOutHasOccurred,
		//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer: nil,
		SendID:                         "7a1aab65-93ab-4f59-b341-3b8fe16f6631",
		MessageInitiatedFromPubSubSend: false,
	}

	// Send message on TimeOutEngineChannel to get information about if TestInstructionExecution already has TimedOut
	*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrack] <- tempTimeOutChannelCommand

	// Response from TimeOutEngine
	var timeOutReturnChannelForTimeOutHasOccurredValue common_config.TimeOutResponseChannelForTimeOutHasOccurredStruct

	// Wait for response from TimeOutEngine
	timeOutReturnChannelForTimeOutHasOccurredValue = <-timeOutResponseChannelForTimeOutHasOccurred

	// Verify that TestInstructionExecution hasn't TimedOut yet
	if timeOutReturnChannelForTimeOutHasOccurredValue.TimeOutWasTriggered == true {
		// TestInstructionExecution had already TimedOut

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_UNSPECIFIED
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     fmt.Sprintf("TestInstructionExecution, '%s' had already TimedOut", finalTestInstructionExecutionResultMessage),
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	if finalTestInstructionExecutionResultMessage.TestInstructionExecutionStatus < fenixExecutionServerGrpcApi.
		TestInstructionExecutionStatusEnum_TIE_CONTROLLED_INTERRUPTION {

		common_config.Logger.WithFields(logrus.Fields{
			"id": "d9ef51cf-1d36-4df2-a719-c1390823e252",
			"finalTestInstructionExecutionResultMessage.TestInstructionExecutionStatus": finalTestInstructionExecutionResultMessage.TestInstructionExecutionStatus,
		}).Error("'TestInstructionExecutionStatus' is not a final status for a TestInstructionExecution. Must be '> %s'", fenixExecutionServerGrpcApi.
			TestInstructionExecutionStatusEnum_TIE_EXECUTING)

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_UNSPECIFIED
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     fmt.Sprintf("'TestInstructionExecutionStatus' is not a final status for a TestInstructionExecution. Got '%s' but expected value '> %s'", finalTestInstructionExecutionResultMessage.TestInstructionExecutionStatus, fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_INITIATED),
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"id":    "76a47577-da52-4cae-82fb-37f0947ad6a9",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'prepareReportCompleteTestInstructionExecutionResultSaveToCloudDB'")

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     "Problem when saving to database",
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Standard is to do a Rollback
	doCommitNotRoleBack = false

	// TestCaseExecutionUuid and TestCaseExecutionVersion based on FinalTestInstructionExecutionResultMessage
	var testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct

	// TestInstructionExecution didn't end with an OK(4, 'TIE_FINISHED_OK' or 5, 'TIE_FINISHED_OK_CAN_BE_RERUN') then Stop further processing
	var thereExistsOnGoingTestInstructionExecutionsOnQueue bool

	// If this is the last TestInstructionExecution and any TestInstructionExecution failed, then trigger change in TestCaseExecutionUuid-status
	var triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutions bool
	var testInstructionExecutionMessageToBroadcastSystem broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct

	defer executionEngine.commitOrRoleBackReportCompleteTestInstructionExecutionResult(
		&txn,
		&doCommitNotRoleBack,
		&testCaseExecutionsToProcess,
		&thereExistsOnGoingTestInstructionExecutionsOnQueue,
		&triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutions,
		&testInstructionExecutionMessageToBroadcastSystem) //txn.Commit(context.Background())

	// Extract TestCaseExecutionQueue-messages to be added to data for ongoing Executions
	err = executionEngine.updateStatusOnTestInstructionsExecutionInCloudDB2(txn, finalTestInstructionExecutionResultMessage)
	if err != nil {

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     "Problem when Updating TestInstructionExecutionStatus in database: " + err.Error(),
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// Load TestCaseExecutionUuid and TestCaseExecutionVersion based on FinalTestInstructionExecutionResultMessage
	testCaseExecutionsToProcess, err = executionEngine.loadTestCaseExecutionAndTestCaseExecutionVersion(txn, finalTestInstructionExecutionResultMessage)
	if err != nil {

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     "Problem when loading TestCaseExecutionUuid and TestCaseExecutionVersion based on FinalTestInstructionExecutionResultMessage, from database: " + err.Error(),
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// Create the BroadCastMessage for the TestInstructionExecution
	var testInstructionExecutionBroadcastMessages []broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct
	testInstructionExecutionBroadcastMessages, err = executionEngine.loadTestInstructionExecutionDetailsForBroadcastMessage(
		txn,
		finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid)

	if err != nil {

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     err.Error(),
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// There should only be one message
	testInstructionExecutionMessageToBroadcastSystem = testInstructionExecutionBroadcastMessages[0]

	// If this is the last on TestInstructionExecution and any of them ended with a 'Non-OK-status' then stop pick new TestInstructionExecutions from Queue
	var testInstructionExecutionSiblingsStatus []*testInstructionExecutionSiblingsStatusStruct
	testInstructionExecutionSiblingsStatus, err = executionEngine.areAllOngoingTestInstructionExecutionsFinishedAndAreAnyTestInstructionExecutionEndedWithNonOkStatus(
		txn, finalTestInstructionExecutionResultMessage)

	if err != nil {

		// Set Error codes to return message
		var errorCodes []fenixExecutionServerGrpcApi.ErrorCodesEnum
		var errorCode fenixExecutionServerGrpcApi.ErrorCodesEnum

		errorCode = fenixExecutionServerGrpcApi.ErrorCodesEnum_ERROR_DATABASE_PROBLEM
		errorCodes = append(errorCodes, errorCode)

		// Create Return message
		ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
			AckNack:                      false,
			Comments:                     "Problem when Checking Database for ongoing end NonOKExecutions: " + err.Error(),
			ErrorCodes:                   errorCodes,
			ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
		}

		return ackNackResponse
	}

	// When there are TestInstructionExecutions in result set then they can have been ended with a Non-OK-status or that they are ongoing in their executions
	if len(testInstructionExecutionSiblingsStatus) != 0 {

		for _, testInstructionExecution := range testInstructionExecutionSiblingsStatus {
			// Is this any ongoing TestInstructionExecutions?
			if testInstructionExecution.testInstructionExecutionStatus < int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_CONTROLLED_INTERRUPTION) {
				thereExistsOnGoingTestInstructionExecutionsOnQueue = true
				break
			}
		}

	} else {
		// All TestInstructionsExecution ended with an OK-status so Update TestCaseExecutionStatus and Check for New TestInstructionExecutions on Queue
		triggerSetTestCaseExecutionStatusAndCheckQueueForNewTestInstructionExecutions = true
	}

	// Update Status on TestCaseExecutionUuid

	// Commit every database change
	doCommitNotRoleBack = true

	// Create Return message
	ackNackResponse = &fenixExecutionServerGrpcApi.AckNackResponse{
		AckNack:                      true,
		Comments:                     "",
		ErrorCodes:                   nil,
		ProtoFileVersionUsedByClient: fenixExecutionServerGrpcApi.CurrentFenixExecutionServerProtoFileVersionEnum(common_config.GetHighestFenixExecutionServerProtoFileVersion()),
	}

	return ackNackResponse
}

type currentTestCaseExecutionStruct struct {
	testCaseExecutionUuid    string
	testCaseExecutionVersion int
}

type testInstructionExecutionSiblingsStatusStruct struct {
	testCaseExecutionUuid                      string
	testCaseExecutionVersion                   int
	testInstructionExecutionUuid               string
	testInstructionInstructionExecutionVersion int
	testInstructionExecutionStatus             int
}

// Update status, which came from Connector/Worker, on ongoing TestInstructionExecution
func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestInstructionsExecutionInCloudDB2(dbTransaction pgx.Tx, finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (err error) {

	// If there are nothing to update then just exit
	if finalTestInstructionExecutionResultMessage == nil {
		return nil
	}

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	testInstructionExecutionUuid := finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid

	var testInstructionExecutionStatus string
	testInstructionExecutionStatus = strconv.Itoa(int(finalTestInstructionExecutionResultMessage.TestInstructionExecutionStatus))
	testInstructionExecutionEndTimeStamp := common_config.ConvertGrpcTimeStampToStringForDB(finalTestInstructionExecutionResultMessage.TestInstructionExecutionEndTimeStamp)

	// Create Update Statement  TestInstructionExecution

	sqlToExecute = sqlToExecute + "UPDATE \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" "
	sqlToExecute = sqlToExecute + fmt.Sprintf("SET ")
	sqlToExecute = sqlToExecute + "\"TestInstructionExecutionStatus\" = " + testInstructionExecutionStatus + ", " // TIE_EXECUTING
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionStatusUpdateTimeStamp\" = '%s', ", currentDataTimeStamp)
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionHasFinished\" = '%s', ", "true")
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionEndTimeStamp\" = '%s' ", testInstructionExecutionEndTimeStamp)
	sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestInstructionExecutionUuid\" = '%s' ", testInstructionExecutionUuid)
	sqlToExecute = sqlToExecute + fmt.Sprintf("AND ")
	sqlToExecute = sqlToExecute + fmt.Sprintf("(\"TestInstructionExecutionStatus\" <> %s ",
		strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_TIMEOUT_INTERRUPTION_CAN_BE_RERUN)))
	sqlToExecute = sqlToExecute + fmt.Sprintf("OR ")
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionStatus\" <> %s) ",
		strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_TIMEOUT_INTERRUPTION)))

	sqlToExecute = sqlToExecute + "; "

	// If no positive responses the just exit
	if len(sqlToExecute) == 0 {
		return nil
	}

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "32cc50e1-b8b1-40a0-bb0b-5e9fbf8dce23",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'updateStatusOnTestInstructionsExecutionInCloudDB2'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "e2a88e5e-a3b0-47d4-b867-93324126fbe7",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	common_config.Logger.WithFields(logrus.Fields{
		"Id":                       "ffa5c358-ba6c-47bd-a828-d9e5d826f913",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// If No(zero) rows were affected then TestInstructionExecutionUuid is missing in Table
	if comandTag.RowsAffected() != 1 {
		errorId := "89e28340-64cd-40f3-921f-caa7729c5d0b"
		err = errors.New(fmt.Sprintf("TestInstructionExecutionUuid '%s' is missing in Table [ErroId: %s]", testInstructionExecutionUuid, errorId))

		common_config.Logger.WithFields(logrus.Fields{
			"Id":                           "e2a88e5e-a3b0-47d4-b867-93324126fbe7",
			"testInstructionExecutionUuid": testInstructionExecutionUuid,
			"sqlToExecute":                 sqlToExecute,
		}).Error("TestInstructionExecutionUuid is missing in Table")

		return err
	}

	// No errors occurred
	return err

}

// Load TestCaseExecutionUuid and TestCaseExecutionVersion based on FinalTestInstructionExecutionResultMessage
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestCaseExecutionAndTestCaseExecutionVersion(
	dbTransaction pgx.Tx,
	finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct, err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\" "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestInstructionExecutionUuid\" = '" + finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid + "'; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "4169f712-36dc-42dd-a9dc-5c8503d8f373",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadTestCaseExecutionAndTestCaseExecutionVersion'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "7291dbb2-7ee7-459d-ba4c-41f9634ccc85",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Variable to store TestCaseExecutionUUID and its TestCaseExecutionVersion
	var channelCommandTestCaseExecutions []ChannelCommandTestCaseExecutionStruct

	// Extract data from DB result set
	for rows.Next() {
		var channelCommandTestCaseExecution ChannelCommandTestCaseExecutionStruct

		err := rows.Scan(
			&channelCommandTestCaseExecution.TestCaseExecutionUuid,
			&channelCommandTestCaseExecution.TestCaseExecutionVersion,
		)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "ab5ec697-c33e-49d1-8f03-e297a05ffccc",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Add TestCaseExecutionUUID and its TestCaseExecutionVersion to slice of messages
		channelCommandTestCaseExecutions = append(channelCommandTestCaseExecutions, channelCommandTestCaseExecution)

	}

	// Verify that we got exactly one row from database
	if len(channelCommandTestCaseExecutions) != 1 {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":                               "84270631-b49c-486a-9ea9-704979d6b387",
			"channelCommandTestCaseExecutions": channelCommandTestCaseExecutions,
			"Number of Rows":                   len(channelCommandTestCaseExecutions),
		}).Error("The result gave not exactly one row from database")

	}

	return channelCommandTestCaseExecutions, err

}

// Verify if siblings to current finished TestInstructionExecutions are all finished and if any of them ended with a Non-OK-status
func (executionEngine *TestInstructionExecutionEngineStruct) areAllOngoingTestInstructionExecutionsFinishedAndAreAnyTestInstructionExecutionEndedWithNonOkStatus(
	dbTransaction pgx.Tx,
	finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage) (
	testInstructionExecutionSiblingsStatus []*testInstructionExecutionSiblingsStatusStruct, err error) {

	// Generate UUID as part of name for Temp-table AND
	//tempTableUuid := uuidGenerator.New().String()
	//tempTableUuidNoDashes := strings.ReplaceAll(tempTableUuid, "-", "")
	//tempTableName := "tempTable_" + tempTableUuidNoDashes

	//usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	// Create SQL that only List TestInstructionExecutions that did not end with a OK-status
	sqlToExecute_part1 := ""

	sqlToExecute_part1 = sqlToExecute_part1 + "SELECT TIUE.\"TestCaseExecutionUuid\",  TIUE.\"TestCaseExecutionVersion\" "
	sqlToExecute_part1 = sqlToExecute_part1 + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute_part1 = sqlToExecute_part1 + "WHERE TIUE.\"TestInstructionExecutionUuid\" = '" +
		finalTestInstructionExecutionResultMessage.TestInstructionExecutionUuid + "' AND "
	sqlToExecute_part1 = sqlToExecute_part1 + "TIUE.\"TestInstructionInstructionExecutionVersion\" = 1; "

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute_part1)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "a239085a-cc01-484e-b023-436c32717a43",
			"Error":        err,
			"sqlToExecute": sqlToExecute_part1,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	var currentTestCaseExecution currentTestCaseExecutionStruct
	var currentTestCaseExecutions []currentTestCaseExecutionStruct
	// Extract data from DB result
	for rows.Next() {

		err = rows.Scan(
			&currentTestCaseExecution.testCaseExecutionUuid,
			&currentTestCaseExecution.testCaseExecutionVersion,
		)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "0c30827d-e9e1-4962-b28b-ea74b05e4dc7",
				"Error":        err,
				"sqlToExecute": sqlToExecute_part1,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Add TestCaseExecutionUuid to slice
		currentTestCaseExecutions = append(currentTestCaseExecutions, currentTestCaseExecution)

	}

	// Exact one TestCaseExecutionUuid should be found
	if len(currentTestCaseExecutions) != 1 {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":                        "22a56463-b892-4732-803a-11a69140e555",
			"sqlToExecute":              sqlToExecute_part1,
			"currentTestCaseExecutions": currentTestCaseExecutions,
		}).Error("Did not found exact one TestCaseExecutionUuid")

		err = errors.New("Did not found exact one TestCaseExecutionUuid")

		return nil, err
	}

	var testCaseExecutionVersionAsString string
	testCaseExecutionVersionAsString = strconv.Itoa(currentTestCaseExecution.testCaseExecutionVersion)

	sqlToExecute_part2 := ""
	sqlToExecute_part2 = sqlToExecute_part2 + "SELECT TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\", " +
		"TIUE.\"TestInstructionExecutionUuid\", TIUE.\"TestInstructionInstructionExecutionVersion\", " +
		"TIUE.\"TestInstructionExecutionStatus\" "
	sqlToExecute_part2 = sqlToExecute_part2 + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute_part2 = sqlToExecute_part2 + "WHERE TIUE.\"TestCaseExecutionUuid\" = '" +
		currentTestCaseExecution.testCaseExecutionUuid + "' AND "
	sqlToExecute_part2 = sqlToExecute_part2 + "TIUE.\"TestCaseExecutionVersion\" = " + testCaseExecutionVersionAsString + " AND "
	sqlToExecute_part2 = sqlToExecute_part2 + "(TIUE.\"TestInstructionExecutionStatus\" < " + strconv.Itoa(int(fenixExecutionServerGrpcApi.
		TestInstructionExecutionStatusEnum_TIE_FINISHED_OK)) + " OR "
	sqlToExecute_part2 = sqlToExecute_part2 + "TIUE.\"TestInstructionExecutionStatus\" > " + strconv.Itoa(int(fenixExecutionServerGrpcApi.
		TestInstructionExecutionStatusEnum_TIE_FINISHED_OK_CAN_BE_RERUN)) + ");"

	// Query DB
	// Execute Query CloudDB
	//TODO change so we use the dbTransaction instead so rows will be locked ----- comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)
	rows2, err := dbTransaction.Query(context.Background(), sqlToExecute_part2) // fenixSyncShared.DbPool.Query(context.Background(), sqlToExecute_part2)
	defer rows2.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "a414a9b3-bed8-49ed-9ec4-b2077725f7fd",
			"Error":        err,
			"sqlToExecute": sqlToExecute_part2,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Extract data from DB result
	for rows2.Next() {
		var testInstructionExecutionSiblingStatus testInstructionExecutionSiblingsStatusStruct

		err = rows2.Scan(
			&testInstructionExecutionSiblingStatus.testCaseExecutionUuid,
			&testInstructionExecutionSiblingStatus.testCaseExecutionVersion,
			&testInstructionExecutionSiblingStatus.testInstructionExecutionUuid,
			&testInstructionExecutionSiblingStatus.testInstructionInstructionExecutionVersion,
			&testInstructionExecutionSiblingStatus.testInstructionExecutionStatus,
		)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "0c30827d-e9e1-4962-b28b-ea74b05e4dc7",
				"Error":        err,
				"sqlToExecute": sqlToExecute_part2,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Add status for TestInstructionExecution-sibling to slice
		testInstructionExecutionSiblingsStatus = append(testInstructionExecutionSiblingsStatus, &testInstructionExecutionSiblingStatus)

	}

	return testInstructionExecutionSiblingsStatus, err

}

// Load TestInstructionExecution-details to be sent over Broadcast-system
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestInstructionExecutionDetailsForBroadcastMessage(
	dbTransaction pgx.Tx,
	testInstructionExecutionUuid string) (
	[]broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct, error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\", "
	sqlToExecute = sqlToExecute + "TIUE.\"TestInstructionExecutionUuid\", TIUE.\"TestInstructionInstructionExecutionVersion\", "
	sqlToExecute = sqlToExecute + "TIUE.\"SentTimeStamp\", TIUE.\"ExpectedExecutionEndTimeStamp\", "
	sqlToExecute = sqlToExecute + "TIUE.\"TestInstructionExecutionStatus\", TIUE.\"TestInstructionExecutionEndTimeStamp\", "
	sqlToExecute = sqlToExecute + "TIUE.\"TestInstructionExecutionHasFinished\", TIUE.\"UniqueCounter\", "
	sqlToExecute = sqlToExecute + "TIUE.\"TestInstructionCanBeReExecuted\", TIUE.\"ExecutionStatusUpdateTimeStamp\" "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestInstructionExecutionUuid\" = '" + testInstructionExecutionUuid + "'; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "7e87fd13-e360-4ca3-b1af-3d508a2a1b9d",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'areAllOngoingTestInstructionExecutionsFinishedAndAreAnyTestInstructionExecutionEndedWithNonOkStatus'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "f0fbac73-b7e6-4eea-9932-4ce49d690fd8",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Temporary variables used when scanning from database
	var (
		tempTestCaseExecutionUuid                string
		tempTestCaseExecutionVersion             int
		tempTestInstructionExecutionUuid         string
		tempTestInstructionExecutionVersion      int
		tempSentTimeStamp                        time.Time
		tempExpectedExecutionEndTimeStamp        *time.Time
		tempTestInstructionExecutionStatusValue  int
		tempTestInstructionExecutionEndTimeStamp *time.Time
		tempTestInstructionExecutionHasFinished  bool
		tempUniqueDatabaseRowCounter             int
		tempTestInstructionCanBeReExecuted       bool
		tempExecutionStatusUpdateTimeStamp       time.Time
	)

	// Variable to store Message to be broadcast
	var testInstructionExecutionBroadcastMessages []broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct

	// Extract data from DB result set
	for rows.Next() {
		var tempTestInstructionExecutionBroadcastMessage broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct

		err = rows.Scan(
			&tempTestCaseExecutionUuid,
			&tempTestCaseExecutionVersion,
			&tempTestInstructionExecutionUuid,
			&tempTestInstructionExecutionVersion,
			&tempSentTimeStamp,
			&tempExpectedExecutionEndTimeStamp,
			&tempTestInstructionExecutionStatusValue,
			&tempTestInstructionExecutionEndTimeStamp,
			&tempTestInstructionExecutionHasFinished,
			&tempUniqueDatabaseRowCounter,
			&tempTestInstructionCanBeReExecuted,
			&tempExecutionStatusUpdateTimeStamp,
		)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "ab5ec697-c33e-49d1-8f03-e297a05ffccc",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Handle Null-values
		var tempExpectedExecutionEndTimeStampAsString string
		if tempExpectedExecutionEndTimeStamp == nil {
			tempExpectedExecutionEndTimeStampAsString = ""
		} else {
			tempExpectedExecutionEndTimeStampAsString = common_config.GenerateDatetimeFromTimeInputForDB(*tempExpectedExecutionEndTimeStamp)
		}

		var tempTestInstructionExecutionEndTimeStampAsString string
		if tempTestInstructionExecutionEndTimeStamp == nil {
			tempTestInstructionExecutionEndTimeStampAsString = ""
		} else {
			tempTestInstructionExecutionEndTimeStampAsString = common_config.GenerateDatetimeFromTimeInputForDB(*tempTestInstructionExecutionEndTimeStamp)
		}

		// Convert 'tempVariables' into a 'TestInstructionExecutionBroadcastMessage'
		tempTestInstructionExecutionBroadcastMessage = broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct{
			TestCaseExecutionUuid:           tempTestCaseExecutionUuid,
			TestCaseExecutionVersion:        strconv.Itoa(int(tempTestCaseExecutionVersion)),
			TestInstructionExecutionUuid:    tempTestInstructionExecutionUuid,
			TestInstructionExecutionVersion: strconv.Itoa(int(tempTestInstructionExecutionVersion)),
			SentTimeStamp:                   common_config.GenerateDatetimeFromTimeInputForDB(tempSentTimeStamp),
			ExpectedExecutionEndTimeStamp:   tempExpectedExecutionEndTimeStampAsString,
			TestInstructionExecutionStatusName: fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_name[int32(
				tempTestInstructionExecutionStatusValue)],
			TestInstructionExecutionStatusValue:  strconv.Itoa(int(tempTestInstructionExecutionStatusValue)),
			TestInstructionExecutionEndTimeStamp: tempTestInstructionExecutionEndTimeStampAsString,
			TestInstructionExecutionHasFinished:  strconv.FormatBool(tempTestInstructionExecutionHasFinished),
			UniqueDatabaseRowCounter:             strconv.Itoa(int(tempUniqueDatabaseRowCounter)),
			TestInstructionCanBeReExecuted:       strconv.FormatBool(tempTestInstructionExecutionHasFinished),
			ExecutionStatusUpdateTimeStamp:       common_config.GenerateDatetimeFromTimeInputForDB(tempExecutionStatusUpdateTimeStamp),
		}

		// Add TestCaseExecutionUUID and its TestCaseExecutionVersion to slice of messages
		testInstructionExecutionBroadcastMessages = append(testInstructionExecutionBroadcastMessages, tempTestInstructionExecutionBroadcastMessage)

	}

	// Verify that we got exactly one row from database
	if len(testInstructionExecutionBroadcastMessages) != 1 {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "84270631-b49c-486a-9ea9-704979d6b387",
			"testInstructionExecutionBroadcastMessages": testInstructionExecutionBroadcastMessages,
			"Number of Rows": len(testInstructionExecutionBroadcastMessages),
		}).Error("The result gave not exactly ONE row from database")

		return nil, errors.New("the result gave not exactly ONE row from database")

	}

	return testInstructionExecutionBroadcastMessages, err

}
