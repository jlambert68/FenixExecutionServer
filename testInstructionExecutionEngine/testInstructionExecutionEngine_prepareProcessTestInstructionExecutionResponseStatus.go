package testInstructionExecutionEngine

import (
	"FenixExecutionServer/broadcastingEngine_ExecutionStatusUpdate"
	"FenixExecutionServer/common_config"
	"context"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixExecutionWorkerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionWorkerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

// Prepare for Saving the ongoing Execution of a new TestCaseExecutionUuid in the CloudDB that were received via gRPC
// or picked up from database when this is the one Execution-instance that is responsible for TestInstructionExecution
func (executionEngine *TestInstructionExecutionEngineStruct) prepareProcessTestInstructionExecutionResponseStatusSaveToCloudDB(
	executionTrackNumber int,
	testInstructionExecutionResponseMessage *fenixExecutionServerGrpcApi.ProcessTestInstructionExecutionResponseStatus) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id":                          "0ebf09c9-28c8-4fb3-bc3d-f734a736be9d",
		"testCaseExecutionsToProcess": testInstructionExecutionResponseMessage,
	}).Debug("Incoming 'prepareProcessTestInstructionExecutionResponseStatusSaveToCloudDB'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "b654019a-05dc-4071-b647-96e85a1e0907",
	}).Debug("Outgoing 'prepareProcessTestInstructionExecutionResponseStatusSaveToCloudDB'")

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Begin SQL Transaction
	dbTransaction, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "3de76867-a91b-4d63-8145-2cf32d9b2215",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'prepareProcessTestInstructionExecutionResponseStatusSaveToCloudDB'")

		return
	}

	// All TestInstructionExecutions
	var testInstructionExecutions []broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct

	// All TestCaseExecutions to update status on
	var channelCommandTestCaseExecution []ChannelCommandTestCaseExecutionStruct

	// Defines if a message should be broadcasted for signaling a ExecutionStatus change
	var messageShallBeBroadcasted bool

	// Load TestCaseExecutionUuid and TestCaseExecutionVersion based on TestInstructionExecutionResponseMessage
	channelCommandTestCaseExecution, err = executionEngine.
		loadTestCaseExecutionAndTestCaseExecutionVersionBasedOnTestInstructionExecutionResponseMessage(
			dbTransaction,
			testInstructionExecutionResponseMessage)

	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "4d3790eb-3a2e-4242-91db-8a777bc7a68c",
			"error": err,
		}).Error("Couldn't Load TestCaseExecutionUuid and TestCaseExecutionVersion based on TestInstructionExecutionResponseMessage")

		return
	}

	// Define Execution Track based on "lowest "TestCaseExecutionUuid
	executionTrackNumber = common_config.CalculateExecutionTrackNumber(
		testInstructionExecutionResponseMessage.TestInstructionExecutionUuid)

	// Standard is to do a Rollback
	doCommitNotRoleBack = false
	defer executionEngine.sendNewTestInstructionsThatIsWaitingToBeSentWorkerCommitOrRoleBackParallellSave(
		&executionTrackNumber,
		&dbTransaction,
		&doCommitNotRoleBack,
		&testInstructionExecutions,
		&channelCommandTestCaseExecution,
		&messageShallBeBroadcasted)

	// Update status on TestInstructions that could be sent to workers
	var testInstructionsThatWasSentToWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer
	var testInstructionThatWasSentToWorkersAndTheResponse *processTestInstructionExecutionRequestAndResponseMessageContainer

	// Only set needed values
	testInstructionThatWasSentToWorkersAndTheResponse = &processTestInstructionExecutionRequestAndResponseMessageContainer{
		domainUuid:               "",
		addressToExecutionWorker: "",
		testCaseExecutionUuid:    channelCommandTestCaseExecution[0].TestCaseExecutionUuid,
		testCaseExecutionVersion: int(channelCommandTestCaseExecution[0].TestCaseExecutionVersion),
		processTestInstructionExecutionRequest: &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest{
			ProtoFileVersionUsedByClient: 0,
			TestInstruction: &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionExecutionMessage{
				TestInstructionExecutionUuid: testInstructionExecutionResponseMessage.GetTestInstructionExecutionUuid(),
				TestInstructionUuid:          "",
				TestInstructionName:          "",
				MajorVersionNumber:           0,
				MinorVersionNumber:           0,
				TestInstructionAttributes:    nil,
			},
			TestData: &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestDataMessage{
				TestDataSetUuid:           "",
				ManualOverrideForTestData: nil,
			},
		},
		processTestInstructionExecutionResponse: &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionResponse{
			AckNackResponse: &fenixExecutionWorkerGrpcApi.AckNackResponse{
				AckNack:                      true,
				Comments:                     "",
				ErrorCodes:                   nil,
				ProtoFileVersionUsedByClient: 0,
			},
			TestInstructionExecutionUuid:   testInstructionExecutionResponseMessage.GetTestInstructionExecutionUuid(),
			ExpectedExecutionDuration:      testInstructionExecutionResponseMessage.GetExpectedExecutionDuration(),
			TestInstructionCanBeReExecuted: testInstructionExecutionResponseMessage.GetTestInstructionCanBeReExecuted(),
		},
	}

	// Append the TestInstructionExecution-response to the slice
	testInstructionsThatWasSentToWorkersAndTheResponse = append(
		testInstructionsThatWasSentToWorkersAndTheResponse, testInstructionThatWasSentToWorkersAndTheResponse)

	// Update TestInstructionExecution-status in database
	err = executionEngine.updateStatusOnTestInstructionsExecutionInCloudDB(dbTransaction, testInstructionsThatWasSentToWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "8384079c-7914-43f8-8d89-7fb2df342cb0",
			"error": err,
		}).Error("Couldn't update TestInstructionExecutionStatus in CloudDB")

		return

	}

	// Update status on TestCases that TestInstructions have been sent to workers
	err = executionEngine.updateStatusOnTestCasesExecutionInCloudDB(dbTransaction, testInstructionsThatWasSentToWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "49a3f6ef-5769-49c4-9738-12d80c69ce51",
			"error": err,
		}).Error("Couldn't TestCaseExecutionStatus in CloudDB")

		return

	}

	// Prepare message data to be sent over Broadcast system
	for _, testInstructionExecutionStatusMessage := range testInstructionsThatWasSentToWorkersAndTheResponse {

		// Create the BroadCastMessage for the TestInstructionExecution
		var testInstructionExecutionBroadcastMessages []broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct
		testInstructionExecutionBroadcastMessages, err = executionEngine.loadTestInstructionExecutionDetailsForBroadcastMessage(
			dbTransaction,
			testInstructionExecutionStatusMessage.processTestInstructionExecutionResponse.TestInstructionExecutionUuid)

		if err != nil {

			return
		}

		// There should only be one message
		var testInstructionExecutionDetailsForBroadcastSystem broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct
		testInstructionExecutionDetailsForBroadcastSystem = testInstructionExecutionBroadcastMessages[0]

		var tempTestInstructionExecutionStatus fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum

		// Only BroadCast 'TIE_EXECUTING' if we got an AckNack=true as respons
		if testInstructionExecutionStatusMessage.processTestInstructionExecutionResponse.AckNackResponse.AckNack == true {
			testInstructionExecutionDetailsForBroadcastSystem.TestInstructionExecutionStatusName =
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_name[int32(
					fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_EXECUTING)]
			testInstructionExecutionDetailsForBroadcastSystem.TestInstructionExecutionStatusName =
				strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_EXECUTING))

			tempTestInstructionExecutionStatus = fenixExecutionServerGrpcApi.
				TestInstructionExecutionStatusEnum_TIE_EXECUTING

		} else {
			//  BroadCast 'TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN' if we got an AckNack=false as respons
			testInstructionExecutionDetailsForBroadcastSystem.TestInstructionExecutionStatusName =
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_name[int32(
					fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN)]
			testInstructionExecutionDetailsForBroadcastSystem.TestInstructionExecutionStatusName =
				strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN))

			tempTestInstructionExecutionStatus = fenixExecutionServerGrpcApi.
				TestInstructionExecutionStatusEnum_TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN
		}

		// Should the message be broadcasted
		messageShallBeBroadcasted = executionEngine.shouldMessageBeBroadcasted(
			shouldMessageBeBroadcasted_ThisIsATestInstructionExecution,
			testInstructionExecutionBroadcastMessages[0].ExecutionStatusReportLevel,
			fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TestInstructionExecutionStatusEnum_DEFAULT_NOT_SET),
			tempTestInstructionExecutionStatus)

		// Add TestInstructionExecution to slice of executions to be sent over Broadcast system
		testInstructionExecutions = append(testInstructionExecutions, testInstructionExecutionDetailsForBroadcastSystem)

		// Make the list of TestCaseExecutions to be able to update Status on TestCaseExecutions
		var tempTestCaseExecutionsMap map[string]string
		var tempTestCaseExecutionsMapKey string
		var existInMap bool
		tempTestCaseExecutionsMap = make(map[string]string)

		// Create the MapKey used for Map
		tempTestCaseExecutionsMapKey = testInstructionExecutionStatusMessage.testCaseExecutionUuid + strconv.Itoa(testInstructionExecutionStatusMessage.testCaseExecutionVersion)
		_, existInMap = tempTestCaseExecutionsMap[tempTestCaseExecutionsMapKey]

		if existInMap == false {
			// Only add to slice when it's a new TestCaseExecutions not handled yet in Map
			var newTempChannelCommandTestCaseExecution ChannelCommandTestCaseExecutionStruct
			newTempChannelCommandTestCaseExecution = ChannelCommandTestCaseExecutionStruct{
				TestCaseExecutionUuid:    testInstructionExecutionStatusMessage.testCaseExecutionUuid,
				TestCaseExecutionVersion: int32(testInstructionExecutionStatusMessage.testCaseExecutionVersion),
			}

			channelCommandTestCaseExecution = append(channelCommandTestCaseExecution, newTempChannelCommandTestCaseExecution)

			// Add to map that this TestCaseExecution has been added to slice
			tempTestCaseExecutionsMap[tempTestCaseExecutionsMapKey] = tempTestCaseExecutionsMapKey
		}

	}

	// Set TimeOut-timers for TestInstructionExecutions in TimerOutEngine if we got an AckNack=true as respons
	err = executionEngine.setTimeOutTimersForTestInstructionExecutions(testInstructionsThatWasSentToWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "a2e710d6-c17b-4da2-8d53-f614cd9986fb",
			"error": err,
		}).Error("Problem when sending TestInstructionExecutions to TimeOutEngine")

		return

	}

	//Remove Allocation for TimeOut-timer because we got an AckNack=false as respons
	err = executionEngine.removeTimeOutTimerAllocationsForTestInstructionExecutions(testInstructionsThatWasSentToWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "60c2975c-c177-413b-a908-814fb2472f41",
			"error": err,
		}).Error("Remove Allocation for TimeOut-timer")

		return

	}

	// Commit every database change
	doCommitNotRoleBack = true

	return
}

// Load TestCaseExecutionUuid and TestCaseExecutionVersion based on TestInstructionExecutionResponseMessage
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestCaseExecutionAndTestCaseExecutionVersionBasedOnTestInstructionExecutionResponseMessage(
	dbTransaction pgx.Tx,
	testInstructionExecutionResponseMessage *fenixExecutionServerGrpcApi.ProcessTestInstructionExecutionResponseStatus) (
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct, err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\", " +
		"TIUE.\"ExecutionStatusReportLevel\"  "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestInstructionExecutionUuid\" = '" + testInstructionExecutionResponseMessage.TestInstructionExecutionUuid + "'; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "a6cacc53-5a09-456c-9fcf-8541184027d0",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadTestCaseExecutionAndTestCaseExecutionVersionBasedOnTestInstructionExecutionResponseMessage'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "ca58f1e2-44d0-44f3-81a3-e9f92f073c4f",
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
			&channelCommandTestCaseExecution.ExecutionStatusReportLevelEnum,
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
