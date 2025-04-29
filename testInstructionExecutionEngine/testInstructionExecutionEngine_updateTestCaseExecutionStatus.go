package testInstructionExecutionEngine

import (
	"FenixExecutionServer/broadcastingEngine_ExecutionStatusUpdate"
	"FenixExecutionServer/common_config"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixExecutionServerGuiGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGuiGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strconv"
	"strings"
	"time"
)

func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestCaseExecutionInCloudDBCommitOrRoleBack(
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool,
	//testCaseExecutionsToProcessReference *[]ChannelCommandTestCaseExecutionStruct,
	testCaseExecutionsReference *[]broadcastingEngine_ExecutionStatusUpdate.TestCaseExecutionBroadcastMessageStruct) {

	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference
	//testCaseExecutionsToProcess := *testCaseExecutionsToProcessReference
	testCaseExecutions := *testCaseExecutionsReference

	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

		// Only try to broadcast if there are messages to send
		if len(testCaseExecutions) > 0 {

			// Create message to be sent to BroadcastEngine
			var broadcastingMessageForExecutions broadcastingEngine_ExecutionStatusUpdate.BroadcastingMessageForExecutionsStruct
			broadcastingMessageForExecutions = broadcastingEngine_ExecutionStatusUpdate.BroadcastingMessageForExecutionsStruct{
				OriginalMessageCreationTimeStamp: strings.Split(time.Now().UTC().String(), " m=")[0],
				TestCaseExecutions:               testCaseExecutions,
				TestInstructionExecutions:        nil,
			}

			// Send message to BroadcastEngine over channel
			broadcastingEngine_ExecutionStatusUpdate.BroadcastEngineMessageChannel <- broadcastingMessageForExecutions

			defer executionEngine.logger.WithFields(logrus.Fields{
				"id":                               "6ad2a565-bd85-4e69-a677-9212beddd94f",
				"broadcastingMessageForExecutions": broadcastingMessageForExecutions,
			}).Debug("Sent message on Broadcast channel")
		}
		/*
			// Trigger TestInstructionEngine to check if there are any TestInstructions on the ExecutionQueue
			go func() {
				channelCommandMessage := ChannelCommandStruct{
					ChannelCommand:                   ChannelCommandCheckForTestInstructionExecutionWaitingOnQueue,
					ChannelCommandTestCaseExecutions: testCaseExecutionsToProcess,
				}

				*executionEngine.executionEngineChannelRef <- channelCommandMessage



			}()

		*/
	} else {
		dbTransaction.Rollback(context.Background())
	}
}

// Update status, which came from Connector/Worker, on ongoing TestCaseExecutionUuid
func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestCaseExecutionInCloudDB(
	executionTrackNumber int,
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id":                          "c11ebfee-0ca3-4288-af46-7fa9bbe26faa",
		"testCaseExecutionsToProcess": testCaseExecutionsToProcess,
	}).Debug("Incoming 'updateStatusOnTestCaseExecutionInCloudDB'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "c813ca2d-f4f2-4148-b800-662622999a5f",
	}).Debug("Outgoing 'updateStatusOnTestCaseExecutionInCloudDB'")

	// If there are nothing to update then just exit
	if len(testCaseExecutionsToProcess) == 0 {
		return nil
	}

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "01615e5a-2410-4233-b234-0c083ca561ab",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'updateStatusOnTestCaseExecutionInCloudDB'")

		return err
	}

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Standard is to do a Rollback
	doCommitNotRoleBack = false

	// All TestCaseExecutions
	var testCaseExecutions []broadcastingEngine_ExecutionStatusUpdate.TestCaseExecutionBroadcastMessageStruct

	defer executionEngine.updateStatusOnTestCaseExecutionInCloudDBCommitOrRoleBack(
		&txn,
		&doCommitNotRoleBack,
		//&testCaseExecutionsToProcess,
		&testCaseExecutions)

	// Load status for each TestInstructionExecution to be able to set correct status on the corresponding TestCaseExecutionUuid
	var loadTestInstructionExecutionStatusMessages []*loadTestInstructionExecutionStatusMessagesStruct
	loadTestInstructionExecutionStatusMessages, err = executionEngine.loadTestInstructionExecutionStatusMessages(
		txn, testCaseExecutionsToProcess)

	// Exit when there was a problem reading  the database
	if err != nil {
		return err
	}

	// Transform TestInstructionExecutionStatus into correct prioritized TestCaseExecutionStatus
	var testCaseExecutionStatusMessages []*testCaseExecutionStatusStruct
	testCaseExecutionStatusMessages, err = executionEngine.transformTestInstructionExecutionStatusIntoTestCaseExecutionStatus(
		loadTestInstructionExecutionStatusMessages)

	// Exit when there was a problem when converting into TestCaseExecutionStatus
	if err != nil {
		return err
	}

	// Get number of TestInstructionExecutions that is waiting on TestInstructionExecutionQueue
	var numberOfTestInstructionExecutionsOnExecutionQueueMap numberOfTestInstructionExecutionsOnQueueMapType
	numberOfTestInstructionExecutionsOnExecutionQueueMap, err = executionEngine.loadNumberOfTestInstructionExecutionsOnExecutionQueue(
		txn, testCaseExecutionsToProcess)
	// Exit when there was a problem updating the database
	if err != nil {
		return err
	}

	// Update TestExecutions in database with the new TestCaseExecutionStatus
	err = executionEngine.updateTestCaseExecutionsWithNewTestCaseExecutionStatus(
		txn, testCaseExecutionStatusMessages, numberOfTestInstructionExecutionsOnExecutionQueueMap)

	// Exit when there was a problem updating the database
	if err != nil {
		return err
	}

	// Prepare message data to be sent over Broadcast system
	for _, testCaseExecutionStatusMessage := range testCaseExecutionStatusMessages {

		// Should the message be broadcasted
		var messageShouldBeBroadcasted bool
		messageShouldBeBroadcasted = executionEngine.shouldMessageBeBroadcasted(
			shouldMessageBeBroadcasted_ThisIsATestCaseExecution,
			testCaseExecutionStatusMessage.ExecutionStatusReportLevel,
			fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum(testCaseExecutionStatusMessage.TestCaseExecutionStatus),
			fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TestInstructionExecutionStatusEnum_DEFAULT_NOT_SET))

		if messageShouldBeBroadcasted == true {

			var testCaseExecution broadcastingEngine_ExecutionStatusUpdate.TestCaseExecutionBroadcastMessageStruct
			testCaseExecution = broadcastingEngine_ExecutionStatusUpdate.TestCaseExecutionBroadcastMessageStruct{
				TestCaseExecutionUuid:    testCaseExecutionStatusMessage.TestCaseExecutionUuid,
				TestCaseExecutionVersion: strconv.Itoa(testCaseExecutionStatusMessage.TestCaseExecutionVersion),
				TestCaseExecutionStatus: fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_name[int32(
					testCaseExecutionStatusMessage.TestCaseExecutionStatus)],

				ExecutionStartTimeStamp: testCaseExecutionStatusMessage.ExecutionStartTimeStampAsString,

				ExecutionHasFinished:           testCaseExecutionStatusMessage.ExecutionHasFinishedAsString,
				ExecutionStatusUpdateTimeStamp: testCaseExecutionStatusMessage.ExecutionStatusUpdateTimeStampAsString,
			}

			// Only send 'ExecutionStopTimeStamp' when the TestCaseExecutionUuid is finished
			if testCaseExecutionStatusMessage.ExecutionHasFinishedAsString == "true" {
				testCaseExecution.ExecutionStopTimeStamp = testCaseExecutionStatusMessage.ExecutionStopTimeStampAsString
			}

			// Add TestCaseExecutionUuid to slice of executions to be sent over Broadcast system
			testCaseExecutions = append(testCaseExecutions, testCaseExecution)

		} else {

			// Message should not be broadcasted

		}

	}

	// Loop all 'testCaseExecutionStatusMessages' and check for "End status"
	for _, testCaseExecutionStatusMessage := range testCaseExecutionStatusMessages {

		// If TestCaseExecutionStatus is an "End status" then add all TestInstructionExecutions to table 'TestCasesExecutionsForListings'
		if hasTestCaseAnEndStatus(int32(testCaseExecutionStatusMessage.TestCaseExecutionStatus)) == true {

			var testInstructionsExecutionStatusPreviewValuesMessage *fenixExecutionServerGrpcApi.TestInstructionsExecutionStatusPreviewValuesMessage

			// Load all TestInstructionExecutions for TestCase
			testInstructionsExecutionStatusPreviewValuesMessage, err = executionEngine.
				loadTestInstructionsExecutionStatusPreviewValues(txn, testCaseExecutionStatusMessage)

			// Exit when there was a problem reading the database
			if err != nil {
				return err
			}

			// Load TestCaseExecution-status
			var testCaseExecutionStatus int32
			testCaseExecutionStatus, err = executionEngine.
				loadTestCaseExecutionStatus(txn, testCaseExecutionStatusMessage)

			// Exit when there was a problem reading the database
			if err != nil {
				return err
			}

			// Update status for TestInstructions in table 'TestCasesExecutionsForListings'

			// Add "ExecutionStatusPreviewValues" to 'TestCasesExecutionsForListings'
			err = executionEngine.addExecutionStatusPreviewValuesIntoDatabase(
				txn,
				testCaseExecutionStatusMessage,
				testInstructionsExecutionStatusPreviewValuesMessage,
				testCaseExecutionStatus)

			// Exit when there was a problem updating the database
			if err != nil {
				return err
			}

		}

	}

	// No errors occurred so secure that commit is done
	doCommitNotRoleBack = true

	return err

}

func hasTestCaseAnEndStatus(testCaseExecutionStatus int32) (isTestCaseEndStatus bool) {

	var testCaseExecutionStatusProto fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum
	testCaseExecutionStatusProto = fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum(testCaseExecutionStatus)

	switch testCaseExecutionStatusProto {

	// Is an End status
	case fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_CONTROLLED_INTERRUPTION,
		fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_CONTROLLED_INTERRUPTION_CAN_BE_RERUN,
		fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_OK,
		fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_OK_CAN_BE_RERUN,
		fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_NOT_OK,
		fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_NOT_OK_CAN_BE_RERUN,
		fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_UNEXPECTED_INTERRUPTION,
		fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN:

		isTestCaseEndStatus = true

	// Is not an End status
	default:
		isTestCaseEndStatus = false

	}

	return isTestCaseEndStatus
}

// Used as type for response object when calling 'loadTestInstructionExecutionStatusMessages'
type loadTestInstructionExecutionStatusMessagesStruct struct {
	TestCaseExecutionUuid          string
	TestCaseExecutionVersion       int
	TestInstructionExecutionOrder  int
	TestInstructionName            string
	TestInstructionExecutionStatus int
}

// Used as type when deciding what End-status a TestCaseExecutionUuid should have depending on its TestInstructionExecutions End-status
type testCaseExecutionStatusStruct struct {
	TestCaseExecutionUuid                  string
	TestCaseExecutionVersion               int
	TestCaseExecutionStatus                int
	TestCaseExecutionStatusAsString        string
	ExecutionStartTimeStampAsString        string
	ExecutionStopTimeStampAsString         string
	ExecutionHasFinishedAsString           string
	ExecutionStatusUpdateTimeStampAsString string
	ExecutionStatusReportLevel             fenixExecutionServerGrpcApi.ExecutionStatusReportLevelEnum
}

// used as type when getting number of TestInstructionExecutions on TestInstructionExecutionQueue
type numberOfTestInstructionExecutionsOnQueueStruct struct {
	numberOfTestInstructionExecutionsOnQueue int
	TestInstructionExecutionUuid             string
	TestInstructionExecutionVersion          int
}

// Type used when to store Map with TestInstructionsExecutions from ExecutionQueue
type numberOfTestInstructionExecutionsOnQueueMapType map[string]*numberOfTestInstructionExecutionsOnQueueStruct

// Load status for each TestInstructionExecution to be able to set correct status on the corresponding TestCaseExecutionUuid
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestInstructionExecutionStatusMessages(
	dbTransaction pgx.Tx,
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (
	loadTestInstructionExecutionStatusMessages []*loadTestInstructionExecutionStatusMessagesStruct,
	err error) {

	//usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	// Generate WHERE-values to only target correct 'TestCaseExecutionUuid' together with 'TestCaseExecutionVersion'
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar string
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars string
	for testCaseExecutionCounter, testCaseExecution := range testCaseExecutionsToProcess {
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar =
			"(TIUE.\"TestCaseExecutionUuid\" = '" + testCaseExecution.TestCaseExecutionUuid + "' AND " +
				"TIUE.\"TestCaseExecutionVersion\" = " + strconv.Itoa(int(testCaseExecution.TestCaseExecutionVersion)) + ") "

		switch testCaseExecutionCounter {
		case 0:
			// When this is the first then we need to add 'AND before'
			// *NOT NEEDED* in this Query
			//correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars = "AND "

		default:
			// When this is not the first then we need to add 'OR' after previous
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars =
				correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars + "OR "
		}

		// Add the WHERE-values
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars =
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar

	}

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT TIUE.\"TestCaseExecutionUuid\",  TIUE.\"TestCaseExecutionVersion\", " +
		"TIUE.\"TestInstructionExecutionOrder\", TIUE.\"TestInstructionName\", TIUE.\"TestInstructionExecutionStatus\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "WHERE "
	sqlToExecute = sqlToExecute + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars
	sqlToExecute = sqlToExecute + "ORDER BY TIUE.\"TestCaseExecutionUuid\" ASC, TIUE.\"TestInstructionExecutionOrder\" ASC, " +
		"TIUE.\"TestInstructionName\" ASC "
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "989787b6-d306-4d44-9167-f2c1eae5b701",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadTestInstructionExecutionStatusMessages'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "77d9657d-f784-4adf-a66a-9173f8dc7f08",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return []*loadTestInstructionExecutionStatusMessagesStruct{}, err
	}

	// Extract data from DB result set
	for rows.Next() {
		var loadTestInstructionExecutionStatusMessage loadTestInstructionExecutionStatusMessagesStruct

		err := rows.Scan(
			&loadTestInstructionExecutionStatusMessage.TestCaseExecutionUuid,
			&loadTestInstructionExecutionStatusMessage.TestCaseExecutionVersion,
			&loadTestInstructionExecutionStatusMessage.TestInstructionExecutionOrder,
			&loadTestInstructionExecutionStatusMessage.TestInstructionName,
			&loadTestInstructionExecutionStatusMessage.TestInstructionExecutionStatus,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "fb586bd2-8133-4d90-bb02-f5a36cde32d9",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return []*loadTestInstructionExecutionStatusMessagesStruct{}, err
		}

		// Add TestInstructionExecutionStatus-message to slice of messages
		loadTestInstructionExecutionStatusMessages = append(loadTestInstructionExecutionStatusMessages,
			&loadTestInstructionExecutionStatusMessage)

	}

	return loadTestInstructionExecutionStatusMessages, err

}

// Transform TestInstructionExecutionStatus into correct prioritized TestCaseExecutionStatus
func (executionEngine *TestInstructionExecutionEngineStruct) transformTestInstructionExecutionStatusIntoTestCaseExecutionStatus(testInstructionExecutionStatusMessages []*loadTestInstructionExecutionStatusMessagesStruct) (testCaseExecutionStatusMessages []*testCaseExecutionStatusStruct, err error) {
	// For each combination 'TestCaseExecutionUuid && TestCaseExecutionVersion' create correct TestCaseExecutionStatus
	// Generate Map that decides what Status that 'overwrite' other status
	// (1,  'TIE_INITIATED') -> NOT OK
	// (2,  'TIE_EXECUTING') -> NOT OK
	// (5,  'TIE_FINISHED_OK' -> OK
	// (6,  'TIE_FINISHED_OK_CAN_BE_RERUN' -> OK
	// (8,  'TIE_FINISHED_NOT_OK_CAN_BE_RERUN' -> OK
	// (11, 'TIE_TIMEOUT_INTERRUPTION_CAN_BE_RERUN' -> OK
	// (4,  'TIE_CONTROLLED_INTERRUPTION_CAN_BE_RERUN' -> OK
	// (10,  'TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN' -> OK
	// (7,  'TIE_FINISHED_NOT_OK' -> OK
	// (12, 'TIE_TIMEOUT_INTERRUPTION' -> OK
	// (3,  'TIE_CONTROLLED_INTERRUPTION' -> OK
	// (9,  'TIE_UNEXPECTED_INTERRUPTION' -> OK

	// Inparameter for Map is 'Current Status' and the value for the Map prioritized Order for that Status
	// map[TestInstructionExecutionStatus]=PrioritizationOrder
	var statusOrderDecisionMap = map[int]int{
		1:  1,
		5:  2,
		2:  3,
		6:  4,
		8:  5,
		11: 6,
		4:  7,
		10: 8,
		7:  9,
		12: 10,
		3:  11,
		9:  12}

	var currentTestCaseExecutionUuid string
	var previousTestCaseExecutionUuid string
	var currentTestCaseExecutionVersion int
	var previousTestCaseExecutionVersion int
	var currentStatus int
	var currentStatusOrder int
	var savedStatusOrder int
	var existInMap bool
	type ruleType int
	var ruleResult ruleType
	var foundRule bool
	const (
		ruleFirstTestCaseExecution ruleType = iota
		ruleSameTestCaseExecution
		ruleNewTestCaseExecutionButNotTheFirstOne
	)

	// Loop all TestInstructionExecutions and extract the highest end-status based on rules in 'statusOrderDecisionMap', for each TestCaseExecutionUuid
	for testInstructionExecutionCounter, testInstructionExecution := range testInstructionExecutionStatusMessages {

		currentTestCaseExecutionUuid = testInstructionExecution.TestCaseExecutionUuid
		currentTestCaseExecutionVersion = testInstructionExecution.TestCaseExecutionVersion
		currentStatus = testInstructionExecution.TestInstructionExecutionStatus
		foundRule = false

		//
		currentStatusOrder, existInMap = statusOrderDecisionMap[currentStatus]
		if existInMap == false {
			// Status must exist in map
			executionEngine.logger.WithFields(logrus.Fields{
				"id":            "5f662a03-5fd2-41ec-a4bf-a91f6676aad8",
				"currentStatus": currentStatus,
			}).Error(fmt.Sprintf("Couldn't find TestInstructionExecutionStatus in 'statusOrderDecisionMap'"))

			errorId := "859de457-d6d8-4e23-8691-7581adfb7738"
			err = errors.New(fmt.Sprintf("Couldn't find TestInstructionExecutionStatus, '%s', in 'statusOrderDecisionMap'. [ErrorID='%s']",
				currentStatus, errorId))

			return nil, err
		}

		// First TestCaseExecutionUuid in the list
		if testInstructionExecutionCounter == 0 {

			foundRule = true
			ruleResult = ruleFirstTestCaseExecution
		}

		// Not the first TestCaseExecutionUuid, but the same TestCaseExecutionUuid as previous one
		if foundRule == false &&
			currentTestCaseExecutionUuid == previousTestCaseExecutionUuid &&
			currentTestCaseExecutionVersion == previousTestCaseExecutionVersion {

			foundRule = true
			ruleResult = ruleSameTestCaseExecution
		}

		// Not the first TestCaseExecutionUuid, but a new TestCaseExecutionUuid compared to previous TestCaseExecutionUuid
		if foundRule == false &&
			currentTestCaseExecutionUuid != previousTestCaseExecutionUuid &&
			currentTestCaseExecutionVersion != previousTestCaseExecutionVersion {

			foundRule = true
			ruleResult = ruleNewTestCaseExecutionButNotTheFirstOne
		}

		switch ruleResult {

		case ruleFirstTestCaseExecution:
			// First TestCaseExecutionUuid in the list

			// Create TestCaseExecutionStatus
			var testCaseExecutionStatus *testCaseExecutionStatusStruct
			testCaseExecutionStatus = &testCaseExecutionStatusStruct{
				TestCaseExecutionUuid:    currentTestCaseExecutionUuid,
				TestCaseExecutionVersion: currentTestCaseExecutionVersion,
				TestCaseExecutionStatus:  currentStatus,
			}

			// Add the TestCaseExecutionUuid with status based on status from   first TestInstructionExecution
			testCaseExecutionStatusMessages = append(testCaseExecutionStatusMessages, testCaseExecutionStatus)

		case ruleSameTestCaseExecution:
			// Not the first TestCaseExecutionUuid, but the same TestCaseExecutionUuid as previous one

			// Check if 'currentStatusOrder' is higher than previous saved StatusOrder
			savedStatusOrder, _ = statusOrderDecisionMap[testCaseExecutionStatusMessages[len(testCaseExecutionStatusMessages)-1].TestCaseExecutionStatus]
			if currentStatusOrder > savedStatusOrder {
				// Replace previous Status
				testCaseExecutionStatusMessages[len(testCaseExecutionStatusMessages)-1].TestCaseExecutionStatus = currentStatus
			}

		case ruleNewTestCaseExecutionButNotTheFirstOne:
			// Not the first TestCaseExecutionUuid, but a new TestCaseExecutionUuid compared to previous TestCaseExecutionUuid
			// Create TestCaseExecutionStatus
			var testCaseExecutionStatus *testCaseExecutionStatusStruct
			testCaseExecutionStatus = &testCaseExecutionStatusStruct{
				TestCaseExecutionUuid:    currentTestCaseExecutionUuid,
				TestCaseExecutionVersion: currentTestCaseExecutionVersion,
				TestCaseExecutionStatus:  currentStatus,
			}

			// Add the TestCaseExecutionUuid with status based on status from  new TestInstructionExecution
			testCaseExecutionStatusMessages = append(testCaseExecutionStatusMessages, testCaseExecutionStatus)

		default:
			// Unhandled ruleResult
			errorId := "d3ce2635-cb4e-4574-a4ce-a791bac049ed"

			err = errors.New(fmt.Sprintf("Unhandled ruleResult: '%s'. [ErrorID='%s']", ruleResult, errorId))

			return nil, err

		}

		// Move 'current' values to 'previous'
		previousTestCaseExecutionUuid = currentTestCaseExecutionUuid
		previousTestCaseExecutionVersion = currentTestCaseExecutionVersion
	}

	return testCaseExecutionStatusMessages, err

}

// Load status for each TestInstructionExecution to be able to set correct status on the corresponding TestCaseExecutionUuid
func (executionEngine *TestInstructionExecutionEngineStruct) loadNumberOfTestInstructionExecutionsOnExecutionQueue(dbTransaction pgx.Tx, testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (numberOfTestInstructionExecutionsOnExecutionQueueMap numberOfTestInstructionExecutionsOnQueueMapType, err error) {

	// Initiate response map
	numberOfTestInstructionExecutionsOnExecutionQueueMap = make(numberOfTestInstructionExecutionsOnQueueMapType)
	var mapKey string

	//usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	// Generate WHERE-values to only target correct 'TestCaseExecutionUuid' together with 'TestCaseExecutionVersion'
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar string
	var correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars string
	for testCaseExecutionCounter, testCaseExecution := range testCaseExecutionsToProcess {
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar =
			"(TIEQ.\"TestCaseExecutionUuid\" = '" + testCaseExecution.TestCaseExecutionUuid + "' AND " +
				"TIEQ.\"TestCaseExecutionVersion\" = " + strconv.Itoa(int(testCaseExecution.TestCaseExecutionVersion)) + ") "

		switch testCaseExecutionCounter {
		case 0:
			// When this is the first then we need to add 'AND before'
			// *NOT NEEDED* in this Query
			//correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars = "AND "

		default:
			// When this is not the first then we need to add 'OR' after previous
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars =
				correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars + "OR "
		}

		// Add the WHERE-values
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars =
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar

	}

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT COUNT(TIEQ.*), TIEQ.\"TestCaseExecutionUuid\", TIEQ.\"TestCaseExecutionVersion\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionExecutionQueue\" TIEQ "
	sqlToExecute = sqlToExecute + "WHERE "
	sqlToExecute = sqlToExecute + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars
	sqlToExecute = sqlToExecute + "GROUP BY TIEQ.\"TestCaseExecutionUuid\", TIEQ.\"TestCaseExecutionVersion\" "
	sqlToExecute = sqlToExecute + ";"

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "53223886-a996-4190-8306-5fb06aaaca01",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadNumberOfTestInstructionExecutionsOnExecutionQueue'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "21aa22c2-8b39-4c9c-ace0-3ef8354f61af",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Extract data from DB result set
	for rows.Next() {
		var numberOfTestInstructionExecutionOnExecutionQueue numberOfTestInstructionExecutionsOnQueueStruct

		err := rows.Scan(
			&numberOfTestInstructionExecutionOnExecutionQueue.numberOfTestInstructionExecutionsOnQueue,
			&numberOfTestInstructionExecutionOnExecutionQueue.TestInstructionExecutionUuid,
			&numberOfTestInstructionExecutionOnExecutionQueue.TestInstructionExecutionVersion,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "ac7cd745-63b5-4484-9092-1cb8471d0bb5",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Add TestInstructionExecutionStatus-message to map of messages
		mapKey = numberOfTestInstructionExecutionOnExecutionQueue.TestInstructionExecutionUuid + strconv.Itoa(numberOfTestInstructionExecutionOnExecutionQueue.TestInstructionExecutionVersion)
		numberOfTestInstructionExecutionsOnExecutionQueueMap[mapKey] = &numberOfTestInstructionExecutionOnExecutionQueue

	}

	return numberOfTestInstructionExecutionsOnExecutionQueueMap, err

}

// Update TestExecutions in database with the new TestCaseExecutionStatus
func (executionEngine *TestInstructionExecutionEngineStruct) updateTestCaseExecutionsWithNewTestCaseExecutionStatus(
	dbTransaction pgx.Tx,
	testCaseExecutionStatusMessages []*testCaseExecutionStatusStruct,
	numberOfTestInstructionExecutionsOnExecutionQueueMap numberOfTestInstructionExecutionsOnQueueMapType) (err error) {

	// If there are nothing to update then exit
	if len(testCaseExecutionStatusMessages) == 0 {
		return err
	}

	var mapKey string
	var existsInMap bool

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	// Generate TimeStamp used in SQL
	testCaseExecutionUpdateTimeStamp := common_config.GenerateDatetimeTimeStampForDB()

	// Loop all TestCaseExecutions and execute SQL-Update
	for _, testCaseExecutionStatusMessage := range testCaseExecutionStatusMessages {

		// If there are any TestInstructionExecutions on the ExecutionQueue, then Upgrade status From 'OK', if so is the Case, to Ongoing Execution
		mapKey = testCaseExecutionStatusMessage.TestCaseExecutionUuid + strconv.Itoa(testCaseExecutionStatusMessage.TestCaseExecutionVersion)
		_, existsInMap = numberOfTestInstructionExecutionsOnExecutionQueueMap[mapKey]
		if existsInMap == true {

			// If Current Reported TestCaseStatus is "TCE_FINISHED_OK" or "TCE_FINISHED_OK_CAN_BE_RERUN" then change status to "TCE_EXECUTING"
			if int32(testCaseExecutionStatusMessage.TestCaseExecutionStatus) == int32(fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_OK) ||
				int32(testCaseExecutionStatusMessage.TestCaseExecutionStatus) == int32(fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_FINISHED_OK_CAN_BE_RERUN) {
				testCaseExecutionStatusMessage.TestCaseExecutionStatus = int(fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_EXECUTING)
			}
		}

		var testCaseExecutionStatusAsString string
		var testCaseExecutionVersionAsString string
		var testCaseExecutionHasFinishedAsString string
		testCaseExecutionStatusAsString = strconv.Itoa(testCaseExecutionStatusMessage.TestCaseExecutionStatus)
		testCaseExecutionVersionAsString = strconv.Itoa(testCaseExecutionStatusMessage.TestCaseExecutionVersion)

		// Check if TestCasesExecutionHasFinished or not
		switch testCaseExecutionStatusMessage.TestCaseExecutionStatus {
		case int(fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_INITIATED),
			int(fenixExecutionServerGrpcApi.TestCaseExecutionStatusEnum_TCE_EXECUTING):
			testCaseExecutionHasFinishedAsString = "false"

		default:
			testCaseExecutionHasFinishedAsString = "true"
		}

		//LOCK TABLE 'TestCasesUnderExecution' IN ROW EXCLUSIVE MODE;

		SqlToExecuteRowLock := ""
		SqlToExecuteRowLock = SqlToExecuteRowLock + "SELECT TCEUE.* "
		SqlToExecuteRowLock = SqlToExecuteRowLock + "FROM \"FenixExecution\".\"TestCasesUnderExecution\" TCEUE "
		SqlToExecuteRowLock = SqlToExecuteRowLock + "WHERE TCEUE.\"TestCaseExecutionUuid\" = '" + testCaseExecutionStatusMessage.TestCaseExecutionUuid + "' AND "
		SqlToExecuteRowLock = SqlToExecuteRowLock + "TCEUE.\"TestCaseExecutionVersion\" = " + testCaseExecutionVersionAsString + " "
		SqlToExecuteRowLock = SqlToExecuteRowLock + "FOR UPDATE; "

		// Log SQL to be executed if Environment variable is true
		if common_config.LogAllSQLs == true {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                  "2f275d87-3862-45fb-b39c-10237de31b05",
				"SqlToExecuteRowLock": SqlToExecuteRowLock,
			}).Debug("SQL to be executed within 'updateTestCaseExecutionsWithNewTestCaseExecutionStatus'")
		}

		// Query DB
		var ctx context.Context
		ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer timeOutCancel()

		rows, err := dbTransaction.Query(ctx, SqlToExecuteRowLock)
		defer rows.Close()

		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                  "87de5b59-dd8b-4256-9973-1c422fd890be",
				"Error":               err,
				"SqlToExecuteRowLock": SqlToExecuteRowLock,
			}).Error("Something went wrong when executing SQL")

			return err
		}

		// Variables to used when extract data from result set
		var tempPlacedOnTestExecutionQueueTimeStamp time.Time
		var tempExecutionPriority int

		var tempExecutionStartTimeStamp time.Time
		var tempExecutionStartTimeStampAsString string
		var tempExecutionStopTimeStamp time.Time
		var tempTestCaseExecutionStatus int
		var tempExecutionStatusUpdateTimeStamp time.Time

		var tempUniqueCounter int

		var testCasesUnderExecution []fenixExecutionServerGuiGrpcApi.TestCaseUnderExecutionMessage

		// Extract data from DB result set
		for rows.Next() {

			// Initiate a new variable to store the data
			testCaseUnderExecution := fenixExecutionServerGuiGrpcApi.TestCaseUnderExecutionMessage{}
			testCaseExecutionBasicInformation := fenixExecutionServerGuiGrpcApi.TestCaseExecutionBasicInformationMessage{}
			testCaseExecutionDetails := fenixExecutionServerGuiGrpcApi.TestCaseExecutionDetailsMessage{}

			err := rows.Scan(
				// TestCaseExecutionBasicInformationMessage
				&testCaseExecutionBasicInformation.DomainUuid,
				&testCaseExecutionBasicInformation.DomainName,
				&testCaseExecutionBasicInformation.TestSuiteUuid,
				&testCaseExecutionBasicInformation.TestSuiteName,
				&testCaseExecutionBasicInformation.TestSuiteVersion,
				&testCaseExecutionBasicInformation.TestSuiteExecutionUuid,
				&testCaseExecutionBasicInformation.TestSuiteExecutionVersion,
				&testCaseExecutionBasicInformation.TestCaseUuid,
				&testCaseExecutionBasicInformation.TestCaseName,
				&testCaseExecutionBasicInformation.TestCaseVersion,
				&testCaseExecutionBasicInformation.TestCaseExecutionUuid,
				&testCaseExecutionBasicInformation.TestCaseExecutionVersion,
				&tempPlacedOnTestExecutionQueueTimeStamp,
				&testCaseExecutionBasicInformation.TestDataSetUuid,
				&tempExecutionPriority,

				// TestCaseExecutionDetailsMessage
				&tempExecutionStartTimeStamp,
				&tempExecutionStopTimeStamp,
				&tempTestCaseExecutionStatus,
				&testCaseExecutionDetails.ExecutionHasFinished,
				&tempUniqueCounter,
				&tempExecutionStatusUpdateTimeStamp,

				&testCaseExecutionBasicInformation.ExecutionStatusReportLevel,
			)

			if err != nil {
				common_config.Logger.WithFields(logrus.Fields{
					"Id":                  "a1833e32-5ad5-468d-8e5f-ca6280d58099",
					"Error":               err,
					"SqlToExecuteRowLock": SqlToExecuteRowLock,
				}).Error("Something went wrong when processing result from database")

				return err
			}

			// Convert temp-variables into gRPC-variables
			testCaseExecutionBasicInformation.PlacedOnTestExecutionQueueTimeStamp = timestamppb.New(tempPlacedOnTestExecutionQueueTimeStamp)
			testCaseExecutionBasicInformation.ExecutionPriority = fenixExecutionServerGuiGrpcApi.ExecutionPriorityEnum(tempExecutionPriority)

			testCaseExecutionDetails.ExecutionStartTimeStamp = timestamppb.New(tempExecutionStartTimeStamp)
			testCaseExecutionDetails.ExecutionStopTimeStamp = timestamppb.New(tempExecutionStopTimeStamp)
			testCaseExecutionDetails.TestCaseExecutionStatus = fenixExecutionServerGuiGrpcApi.TestCaseExecutionStatusEnum(tempTestCaseExecutionStatus)
			testCaseExecutionDetails.ExecutionStatusUpdateTimeStamp = timestamppb.New(tempExecutionStatusUpdateTimeStamp)

			// Build 'TestCaseUnderExecutionMessage'
			testCaseUnderExecution.TestCaseExecutionBasicInformation = &testCaseExecutionBasicInformation
			testCaseUnderExecution.TestCaseExecutionDetails = &testCaseExecutionDetails

			// Add 'TestCaseUnderExecutionMessage' to slice of all 'TestCaseUnderExecutionMessage's
			testCasesUnderExecution = append(testCasesUnderExecution, testCaseUnderExecution)

		}

		// If No(zero) rows were affected then TestInstructionExecutionUuid is missing in Table
		if len(testCasesUnderExecution) != 1 {
			errorId := "cc47776a-959f-496a-821b-8a6e6e711549"

			err = errors.New(fmt.Sprintf("TestICaseExecutionUuid '%s' with Execution Version Number '%s' is missing in Table [ErroId: %s]",
				testCaseExecutionStatusMessage.TestCaseExecutionUuid, testCaseExecutionStatusMessage.TestCaseExecutionVersion, errorId))

			common_config.Logger.WithFields(logrus.Fields{
				"Id": "99420b20-0bbc-420d-8123-b7b7e5f8eac7",
				"testCaseExecutionStatusMessage.TestCaseExecutionUuid":    testCaseExecutionStatusMessage.TestCaseExecutionUuid,
				"testCaseExecutionStatusMessage.TestCaseExecutionVersion": testCaseExecutionStatusMessage.TestCaseExecutionVersion,

				"SqlToExecuteRowLock": SqlToExecuteRowLock,
			}).Error("TestInstructionExecutionUuid is missing in Table")

			return err
		}

		// Extract 'ExecutionStartTimeStamp'
		tempExecutionStartTimeStampAsString = testCasesUnderExecution[0].TestCaseExecutionDetails.ExecutionStartTimeStamp.AsTime().String()
		//tempTestCasesExecutionBasicInformation

		// Create Update Statement for each TestCaseExecutionUuid update
		sqlToExecute := ""
		sqlToExecute = sqlToExecute + "UPDATE \"" + usedDBSchema + "\".\"TestCasesUnderExecution\" "
		sqlToExecute = sqlToExecute + fmt.Sprintf("SET ")
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionStopTimeStamp\" = '%s', ", testCaseExecutionUpdateTimeStamp)
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestCaseExecutionStatus\" = %s, ", testCaseExecutionStatusAsString)
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionHasFinished\" = '%s', ", testCaseExecutionHasFinishedAsString)
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionStatusUpdateTimeStamp\" = '%s' ", testCaseExecutionUpdateTimeStamp)
		sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestCaseExecutionUuid\" = '%s' ", testCaseExecutionStatusMessage.TestCaseExecutionUuid)
		sqlToExecute = sqlToExecute + fmt.Sprintf("AND \"TestCaseExecutionVersion\" = %s ", testCaseExecutionVersionAsString)
		sqlToExecute = sqlToExecute + "; "

		// If no positive responses the just exit
		if len(sqlToExecute) == 0 {
			return nil
		}

		// Log SQL to be executed if Environment variable is true
		if common_config.LogAllSQLs == true {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "54fa8563-4b6d-4313-bb11-c63844ab89be",
				"sqlToExecute": sqlToExecute,
			}).Debug("SQL to be executed within 'updateTestCaseExecutionsWithNewTestCaseExecutionStatus'")
		}

		// Execute Query CloudDB
		comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":           "34b9c93b-a94a-4a65-8ccf-041bfae4b250",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when executing SQL")

			return err
		}

		// Log response from CloudDB
		common_config.Logger.WithFields(logrus.Fields{
			"Id":                       "dd728940-319c-466f-9422-373deae31523",
			"comandTag.Insert()":       comandTag.Insert(),
			"comandTag.Delete()":       comandTag.Delete(),
			"comandTag.Select()":       comandTag.Select(),
			"comandTag.Update()":       comandTag.Update(),
			"comandTag.RowsAffected()": comandTag.RowsAffected(),
			"comandTag.String()":       comandTag.String(),
			"sqlToExecute":             sqlToExecute,
		}).Debug("Return data for SQL executed in database")

		// If No(zero) rows were affected then TestInstructionExecutionUuid is missing in Table
		if comandTag.RowsAffected() != 1 {
			errorId := "e3df8260-d463-4644-b999-6e6e94b54956"

			err = errors.New(fmt.Sprintf("TestICaseExecutionUuid '%s' with Execution Version Number '%s' is missing in Table [ErroId: %s]",
				testCaseExecutionStatusMessage.TestCaseExecutionUuid, testCaseExecutionStatusMessage.TestCaseExecutionVersion, errorId))

			common_config.Logger.WithFields(logrus.Fields{
				"Id": "76be13be-8649-41e2-b7f2-3b5e54e26192",
				"testCaseExecutionStatusMessage.TestCaseExecutionUuid":    testCaseExecutionStatusMessage.TestCaseExecutionUuid,
				"testCaseExecutionStatusMessage.TestCaseExecutionVersion": testCaseExecutionStatusMessage.TestCaseExecutionVersion,

				"sqlToExecute": sqlToExecute,
			}).Error("TestInstructionExecutionUuid is missing in Table")

			return err
		}

		// Update TestCaseExecutionDetails to be sent with help of BroadCast-message
		testCaseExecutionStatusMessage.TestCaseExecutionStatusAsString = testCaseExecutionStatusAsString
		testCaseExecutionStatusMessage.ExecutionStartTimeStampAsString = tempExecutionStartTimeStampAsString
		testCaseExecutionStatusMessage.ExecutionStopTimeStampAsString = testCaseExecutionUpdateTimeStamp
		testCaseExecutionStatusMessage.ExecutionStatusUpdateTimeStampAsString = testCaseExecutionUpdateTimeStamp
		testCaseExecutionStatusMessage.ExecutionHasFinishedAsString = testCaseExecutionHasFinishedAsString
		testCaseExecutionStatusMessage.ExecutionStatusReportLevel = fenixExecutionServerGrpcApi.
			ExecutionStatusReportLevelEnum(testCasesUnderExecution[0].GetTestCaseExecutionBasicInformation().
				ExecutionStatusReportLevel)

	}

	// No errors occurred
	return err

}

// Retrieve "ExecutionStatusPreviewValues" for all TestInstructions for one TestCaseExecution
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestInstructionsExecutionStatusPreviewValues(
	dbTransaction pgx.Tx,
	testCaseExecutionStatusMessages *testCaseExecutionStatusStruct) (
	testInstructionsExecutionStatusPreviewValuesMessage *fenixExecutionServerGrpcApi.TestInstructionsExecutionStatusPreviewValuesMessage,
	err error) {

	// Load 'ExecutionStatusPreviewValues'

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\", "
	sqlToExecute = sqlToExecute + "TIUE.\"TestInstructionExecutionUuid\", TIUE.\"TestInstructionInstructionExecutionVersion\", "
	sqlToExecute = sqlToExecute + "TIUE.\"MatureTestInstructionUuid\", TIUE.\"TestInstructionName\", "
	sqlToExecute = sqlToExecute + "TIUE.\"SentTimeStamp\", TIUE.\"TestInstructionExecutionEndTimeStamp\", "
	sqlToExecute = sqlToExecute + "TIUE.\"TestInstructionExecutionStatus\", "
	sqlToExecute = sqlToExecute + "TIUE.\"ExecutionDomainUuid\", TIUE.\"ExecutionDomainName\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "WHERE "
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestCaseExecutionUuid\" = '%s' AND \"TestCaseExecutionVersion\" = %d ",
		testCaseExecutionStatusMessages.TestCaseExecutionUuid,
		testCaseExecutionStatusMessages.TestCaseExecutionVersion)
	sqlToExecute = sqlToExecute + "ORDER BY TIUE.\"SentTimeStamp\" ASC"
	sqlToExecute = sqlToExecute + ";"

	// Query DB
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "0e402c36-1468-459a-b11d-1c43e6995304",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, err
	}

	// Number of rows
	var numberOfRowFromDB int32
	numberOfRowFromDB = 0

	var testCasePreviewAndExecutionStatusPreviewValues []*fenixExecutionServerGrpcApi.TestInstructionExecutionStatusPreviewValueMessage
	var sentTimeStampAsTimeStamp time.Time
	var testInstructionExecutionEndTimeStampAsTimeStamp time.Time
	var nullableTestInstructionExecutionEndTimeStampAsTimeStamp sql.NullTime

	// Extract data from DB result set
	for rows.Next() {

		var testCasePreviewAndExecutionStatusPreviewValue fenixExecutionServerGrpcApi.TestInstructionExecutionStatusPreviewValueMessage
		numberOfRowFromDB = numberOfRowFromDB + 1

		err := rows.Scan(
			&testCasePreviewAndExecutionStatusPreviewValue.TestCaseExecutionUuid,
			&testCasePreviewAndExecutionStatusPreviewValue.TestCaseExecutionVersion,
			&testCasePreviewAndExecutionStatusPreviewValue.TestInstructionExecutionUuid,
			&testCasePreviewAndExecutionStatusPreviewValue.TestInstructionInstructionExecutionVersion,
			&testCasePreviewAndExecutionStatusPreviewValue.MatureTestInstructionUuid,
			&testCasePreviewAndExecutionStatusPreviewValue.TestInstructionName,
			&sentTimeStampAsTimeStamp,
			&nullableTestInstructionExecutionEndTimeStampAsTimeStamp,
			&testCasePreviewAndExecutionStatusPreviewValue.TestInstructionExecutionStatus,
			&testCasePreviewAndExecutionStatusPreviewValue.ExecutionDomainUuid,
			&testCasePreviewAndExecutionStatusPreviewValue.ExecutionDomainName,
		)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"Id":                "e841fb7e-3d00-4a49-829c-391ee7ec7411",
				"Error":             err,
				"sqlToExecute":      sqlToExecute,
				"numberOfRowFromDB": numberOfRowFromDB,
			}).Error("Something went wrong when processing result from database")

			return nil, err
		}

		// Check if the timestamp is valid or NULL
		if nullableTestInstructionExecutionEndTimeStampAsTimeStamp.Valid {
			// Timestamp is not NULL
			testInstructionExecutionEndTimeStampAsTimeStamp = nullableTestInstructionExecutionEndTimeStampAsTimeStamp.Time
		} else {
			// TimeStamp is NULL
			testInstructionExecutionEndTimeStampAsTimeStamp = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

		}

		// Convert DataTime into gRPC-version
		testCasePreviewAndExecutionStatusPreviewValue.SentTimeStamp = timestamppb.New(sentTimeStampAsTimeStamp)
		testCasePreviewAndExecutionStatusPreviewValue.TestInstructionExecutionEndTimeStamp = timestamppb.
			New(testInstructionExecutionEndTimeStampAsTimeStamp)

		// Add value to slice of values
		testCasePreviewAndExecutionStatusPreviewValues = append(testCasePreviewAndExecutionStatusPreviewValues,
			&testCasePreviewAndExecutionStatusPreviewValue)

	}

	testInstructionsExecutionStatusPreviewValuesMessage = &fenixExecutionServerGrpcApi.
		TestInstructionsExecutionStatusPreviewValuesMessage{
		TestInstructionExecutionStatusPreviewValues: testCasePreviewAndExecutionStatusPreviewValues}

	return testInstructionsExecutionStatusPreviewValuesMessage, err

}

// Retrieve "TestCaseExecutionStatus" for one TestCaseExecution
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestCaseExecutionStatus(
	dbTransaction pgx.Tx,
	testCaseExecutionStatusMessages *testCaseExecutionStatusStruct) (
	testCaseExecutionStatus int32,
	err error) {

	// Load 'ExecutionStatusPreviewValues'

	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT TIUE.\"TestCaseExecutionStatus\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestCasesUnderExecution\" TIUE "
	sqlToExecute = sqlToExecute + "WHERE "
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestCaseExecutionUuid\" = '%s' AND \"TestCaseExecutionVersion\" = %d ",
		testCaseExecutionStatusMessages.TestCaseExecutionUuid,
		testCaseExecutionStatusMessages.TestCaseExecutionVersion)
	sqlToExecute = sqlToExecute + ";"

	// Query DB
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "9656c7db-1219-4e3e-b1ca-47e785e669b4",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return 0, err
	}

	// Number of rows
	var numberOfRowFromDB int32
	numberOfRowFromDB = 0

	// Extract data from DB result set
	for rows.Next() {

		numberOfRowFromDB = numberOfRowFromDB + 1

		err := rows.Scan(
			&testCaseExecutionStatus,
		)

		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"Id":                "e841fb7e-3d00-4a49-829c-391ee7ec7411",
				"Error":             err,
				"sqlToExecute":      sqlToExecute,
				"numberOfRowFromDB": numberOfRowFromDB,
			}).Error("Something went wrong when processing result from database")

			return 0, err
		}

	}

	// If number of rows <> 1 then there is a problem
	if numberOfRowFromDB != 1 {

		errorId := "537693e3-7b8e-464e-b272-f3482b315284"

		err = errors.New(fmt.Sprintf("number of rows in database response was not exact 1 row, found %d rows. [ErrorId: %s]",
			numberOfRowFromDB, errorId))

		common_config.Logger.WithFields(logrus.Fields{
			"Id":                "0bf8a0ed-d9c8-4137-97da-6d16e12502d5",
			"Error":             err,
			"sqlToExecute":      sqlToExecute,
			"numberOfRowFromDB": numberOfRowFromDB,
		}).Error("Something went wrong when processing result from database")

		return 0, err
	}

	return testCaseExecutionStatus, err

}

// Add "ExecutionStatusPreviewValues" to 'TestCasesExecutionsForListings'
func (executionEngine *TestInstructionExecutionEngineStruct) addExecutionStatusPreviewValuesIntoDatabase(
	dbTransaction pgx.Tx,
	testCaseExecutionStatusMessages *testCaseExecutionStatusStruct,
	testInstructionsExecutionStatusPreviewValuesMessage *fenixExecutionServerGrpcApi.
		TestInstructionsExecutionStatusPreviewValuesMessage,
	testCaseExecutionStatus int32) (
	err error) {

	// If there are nothing to update then just exit
	if testInstructionsExecutionStatusPreviewValuesMessage == nil {
		return nil
	}

	testInstructionsExecutionStatusPreviewValuesMessageAsJsonb := protojson.Format(testInstructionsExecutionStatusPreviewValuesMessage)

	// Create Update Statement TestCasesExecutionsForListings
	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "UPDATE \"FenixExecution\".\"TestCasesExecutionsForListings\" "
	sqlToExecute = sqlToExecute + fmt.Sprintf("SET ")
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionsExecutionStatusPreviewValues\" = '%s', ",
		testInstructionsExecutionStatusPreviewValuesMessageAsJsonb)
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestCaseExecutionStatus\" = '%d' ",
		testCaseExecutionStatus)
	sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestCaseExecutionUuid\" = '%s' ",
		testCaseExecutionStatusMessages.TestCaseExecutionUuid)
	sqlToExecute = sqlToExecute + fmt.Sprintf("AND ")
	sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestCaseExecutionVersion\" = %d ",
		testCaseExecutionStatusMessages.TestCaseExecutionVersion)
	sqlToExecute = sqlToExecute + "; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "194603bd-9002-45a5-8e8e-147df7439887",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'addExecutionStatusPreviewValuesIntoDatabase'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "7b3f9d20-cc77-4625-b8c1-a334ccad07d4",
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	common_config.Logger.WithFields(logrus.Fields{
		"Id":                       "95c9ff88-4797-4219-8a5d-e6a560b9452c",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// If No(zero) rows were affected then TestInstructionExecutionUuid is missing in Table
	if comandTag.RowsAffected() != 1 {
		errorId := "5d6461e9-3339-4b37-9706-dc0d9c4ea6e1"
		err = errors.New(fmt.Sprintf("TestInstructionExecutionUuid '%s' with TestInstructionExecutionVersion '%d' is missing in Table: 'TestCasesExecutionsForListings' [ErroId: %s]",
			testCaseExecutionStatusMessages.TestCaseExecutionUuid,
			testCaseExecutionStatusMessages.TestCaseExecutionVersion,
			errorId))

		common_config.Logger.WithFields(logrus.Fields{
			"Id":                       "29a366f9-1b1a-4e51-b950-9e64089dc9a3",
			"comandTag.RowsAffected()": comandTag.RowsAffected(),
		}).Error(err.Error())

		return err
	}

	// No errors occurred
	return err

}
