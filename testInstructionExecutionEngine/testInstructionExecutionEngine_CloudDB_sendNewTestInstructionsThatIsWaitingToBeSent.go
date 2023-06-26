package testInstructionExecutionEngine

import (
	"FenixExecutionServer/broadcastingEngine"
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/messagesToExecutionWorker"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	"sort"
	"strings"
	"time"

	//"FenixExecutionServer/testInstructionTimeOutEngine"
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	fenixExecutionWorkerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionWorkerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strconv"
)

func (executionEngine *TestInstructionExecutionEngineStruct) sendNewTestInstructionsThatIsWaitingToBeSentWorkerCommitOrRoleBackParallellSave(
	executionTrackNumberReference *int,
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool,
	testInstructionExecutionsReference *[]broadcastingEngine.TestInstructionExecutionBroadcastMessageStruct,
	channelCommandTestCaseExecutionReference *[]ChannelCommandTestCaseExecutionStruct) {

	executionTrackNumber := *executionTrackNumberReference
	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference
	testInstructionExecutions := *testInstructionExecutionsReference
	channelCommandTestCaseExecution := *channelCommandTestCaseExecutionReference

	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

		// Create message to be sent to BroadcastEngine
		var broadcastingMessageForExecutions broadcastingEngine.BroadcastingMessageForExecutionsStruct
		broadcastingMessageForExecutions = broadcastingEngine.BroadcastingMessageForExecutionsStruct{
			OriginalMessageCreationTimeStamp: strings.Split(time.Now().UTC().String(), " m=")[0],
			TestCaseExecutions:               nil,
			TestInstructionExecutions:        testInstructionExecutions,
		}

		// Send message to BroadcastEngine over channel
		broadcastingEngine.BroadcastEngineMessageChannel <- broadcastingMessageForExecutions

		defer common_config.Logger.WithFields(logrus.Fields{
			"id":                               "dc384f61-13a6-4f3b-9e1d-345669cf3947",
			"broadcastingMessageForExecutions": broadcastingMessageForExecutions,
		}).Debug("Sent message on Broadcast channel")

		// Update status for TestCaseExecutionUuid, based there are TestInstructionExecution
		if len(testInstructionExecutions) > 0 {

			channelCommandMessage := ChannelCommandStruct{
				ChannelCommand:                    ChannelCommandUpdateExecutionStatusOnTestCaseExecutionExecutions,
				ChannelCommandTestCaseExecutions:  channelCommandTestCaseExecution,
				ReturnChannelWithDBErrorReference: nil,
			}

			// Send message on ExecutionEngineChannel
			*executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage
		}

	} else {
		dbTransaction.Rollback(context.Background())
	}
}

// Prepare for Saving the ongoing Execution of a new TestCaseExecutionUuid in the CloudDB
func (executionEngine *TestInstructionExecutionEngineStruct) sendNewTestInstructionsThatIsWaitingToBeSentWorker(
	executionTrackNumber int,
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id":                          "ffd43120-ad83-4f38-9ff5-877c572cc08e",
		"testCaseExecutionsToProcess": testCaseExecutionsToProcess,
	}).Debug("Incoming 'sendNewTestInstructionsThatIsWaitingToBeSentWorker'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "9787e64a-63c6-46e4-8008-06966a77614e",
	}).Debug("Outgoing 'sendNewTestInstructionsThatIsWaitingToBeSentWorker'")

	// After all stuff is done, then Commit or Rollback depending on result
	var doCommitNotRoleBack bool

	// Begin SQL Transaction
	txn, err := fenixSyncShared.DbPool.Begin(context.Background())
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "2effe457-d6b4-47d6-989c-5b4107e52077",
			"error": err,
		}).Error("Problem to do 'DbPool.Begin'  in 'sendNewTestInstructionsThatIsWaitingToBeSentWorker'")

		return
	}

	// All TestInstructionExecutions
	var testInstructionExecutions []broadcastingEngine.TestInstructionExecutionBroadcastMessageStruct

	// All TestCaseExecutions to update status on
	var channelCommandTestCaseExecution []ChannelCommandTestCaseExecutionStruct

	// Extract "lowest" TestCaseExecutionUuid
	if len(testCaseExecutionsToProcess) > 0 {
		var uuidSlice []string
		for _, uuid := range testCaseExecutionsToProcess {
			uuidSlice = append(uuidSlice, uuid.TestCaseExecutionUuid)
		}
		sort.Strings(uuidSlice)

		// Define Execution Track based on "lowest "TestCaseExecutionUuid
		executionTrackNumber = common_config.CalculateExecutionTrackNumber(uuidSlice[0])
	}

	// Standard is to do a Rollback
	doCommitNotRoleBack = false
	defer executionEngine.sendNewTestInstructionsThatIsWaitingToBeSentWorkerCommitOrRoleBackParallellSave(
		&executionTrackNumber,
		&txn,
		&doCommitNotRoleBack,
		&testInstructionExecutions,
		&channelCommandTestCaseExecution)

	// Generate a new TestCaseExecutionUuid-UUID
	//testCaseExecutionUuid := uuidGenerator.New().String()

	// Generate TimeStamp
	//placedOnTestExecutionQueueTimeStamp := time.Now()

	// Load all TestInstructions and their attributes to be sent to the Executions Workers over gRPC
	rawTestInstructionsToBeSentToExecutionWorkers, rawTestInstructionAttributesToBeSentToExecutionWorkers, err := executionEngine.loadNewTestInstructionToBeSentToExecutionWorkers(
		txn, testCaseExecutionsToProcess)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "25cd9e94-76f6-40ca-8f4c-eed10b618224",
			"error": err,
		}).Error("Got some error when loading the TestInstructionExecutions and Attributes for Executions")

		return
	}

	// If there are no TestInstructions in the 'ongoing' then check if there are any on the TestInstructionExecution-queue
	if rawTestInstructionsToBeSentToExecutionWorkers == nil {

		// Trigger TestInstructionEngine to check if there are more TestInstructions that is waiting to be sent from 'Ongoing'
		channelCommandMessage := ChannelCommandStruct{
			ChannelCommand:                   ChannelCommandCheckForTestInstructionExecutionWaitingOnQueue,
			ChannelCommandTestCaseExecutions: testCaseExecutionsToProcess,
		}

		*executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage

		return
	}

	// Transform Raw TestInstructions and Attributes, from DB, into messages ready to be sent over gRPC to Execution Workers
	testInstructionsToBeSentToExecutionWorkersAndTheResponse, err := executionEngine.transformRawTestInstructionsAndAttributeIntoGrpcMessages(rawTestInstructionsToBeSentToExecutionWorkers, rawTestInstructionAttributesToBeSentToExecutionWorkers) //(txn)
	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "25cd9e94-76f6-40ca-8f4c-eed10b618224",
			"error": err,
		}).Error("Some problem when transforming raw rawTestInstructionsToBeSentToExecutionWorkers- and Attributes-data into gRPC-messages")

		return
	}

	// If there are no TestInstructions then exit
	if rawTestInstructionsToBeSentToExecutionWorkers == nil {

		return
	}

	// Send TestInstructionExecutions with their attributes to correct Execution Worker
	err = executionEngine.sendTestInstructionExecutionsToWorker(testInstructionsToBeSentToExecutionWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "ef815265-3b67-4b59-862f-26d352795c95",
			"error": err,
		}).Error("Got some problem when sending sending TestInstructionExecutions to Worker")

		return

	}

	// Update status on TestInstructions that could be sent to workers
	err = executionEngine.updateStatusOnTestInstructionsExecutionInCloudDB(txn, testInstructionsToBeSentToExecutionWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "bdbc719a-d354-4ebb-9d4c-d002ec55426c",
			"error": err,
		}).Error("Couldn't update TestInstructionExecutionStatus in CloudDB")

		return

	}

	// Update status on TestCases that TestInstructions have been sent to workers
	err = executionEngine.updateStatusOnTestCasesExecutionInCloudDB(txn, testInstructionsToBeSentToExecutionWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "89eb961b-413d-47d1-86b1-8fc1f51c564c",
			"error": err,
		}).Error("Couldn't TestCaseExecutionStatus in CloudDB")

		return

	}

	// Prepare message data to be sent over Broadcast system
	for _, testInstructionExecutionStatusMessage := range testInstructionsToBeSentToExecutionWorkersAndTheResponse {

		// Create the BroadCastMessage for the TestInstructionExecution
		var testInstructionExecutionBroadcastMessages []broadcastingEngine.TestInstructionExecutionBroadcastMessageStruct
		testInstructionExecutionBroadcastMessages, err = executionEngine.loadTestInstructionExecutionDetailsForBroadcastMessage(
			txn,
			testInstructionExecutionStatusMessage.processTestInstructionExecutionResponse.TestInstructionExecutionUuid)

		if err != nil {

			return
		}

		// There should only be one message
		var testInstructionExecutionDetailsForBroadcastSystem broadcastingEngine.TestInstructionExecutionBroadcastMessageStruct
		testInstructionExecutionDetailsForBroadcastSystem = testInstructionExecutionBroadcastMessages[0]

		// Only BroadCast 'TIE_EXECUTING' if we got an AckNack=true as respons
		if testInstructionExecutionStatusMessage.processTestInstructionExecutionResponse.AckNackResponse.AckNack == true {
			testInstructionExecutionDetailsForBroadcastSystem.TestInstructionExecutionStatusName =
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_name[int32(
					fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_EXECUTING)]
			testInstructionExecutionDetailsForBroadcastSystem.TestInstructionExecutionStatusName =
				strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_EXECUTING))
		} else {
			//  BroadCast 'TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN' if we got an AckNack=false as respons
			testInstructionExecutionDetailsForBroadcastSystem.TestInstructionExecutionStatusName =
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_name[int32(
					fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN)]
			testInstructionExecutionDetailsForBroadcastSystem.TestInstructionExecutionStatusName =
				strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN))
		}

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
	err = executionEngine.setTimeOutTimersForTestInstructionExecutions(testInstructionsToBeSentToExecutionWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "7d998314-1a9c-40bb-a7f2-9017b26db58f",
			"error": err,
		}).Error("Problem when sending TestInstructionExecutions to TimeOutEngine")

		return

	}

	//Remove Allocation for TimeOut-timer because we got an AckNack=false as respons
	err = executionEngine.removeTimeOutTimerAllocationsForTestInstructionExecutions(testInstructionsToBeSentToExecutionWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "640602b0-04be-4206-987b-4ccc1cb5d46c",
			"error": err,
		}).Error("Problem when sending TestInstructionExecutions to TimeOutEngine")

		return

	}

	// Commit every database change
	doCommitNotRoleBack = true

	return
}

// Hold one new TestInstruction to be sent to Execution Worker
type newTestInstructionToBeSentToExecutionWorkersStruct struct {
	domainUuid                        string
	domainName                        string
	executionWorkerAddress            string
	testInstructionExecutionUuid      string
	testInstructionOriginalUuid       string
	testInstructionName               string
	testInstructionMajorVersionNumber int
	testInstructionMinorVersionNumber int
	testDataSetUuid                   string
	TestCaseExecutionUuid             string
	testCaseExecutionVersion          int
}

// Hold one new TestInstructionAttribute to be sent to Execution Worker
type newTestInstructionAttributeToBeSentToExecutionWorkersStruct struct {
	testInstructionExecutionUuid     string
	testInstructionAttributeType     int
	testInstructionAttributeUuid     string
	testInstructionAttributeName     string
	attributeValueAsString           string
	attributeValueUuid               string
	testInstructionExecutionTypeUuid string
	testInstructionExecutionTypeName string
}

// Load all New TestInstructions and their attributes to be sent to the Executions Workers over gRPC
func (executionEngine *TestInstructionExecutionEngineStruct) loadNewTestInstructionToBeSentToExecutionWorkers(
	dbTransaction pgx.Tx,
	testCaseExecutionsToProcess []ChannelCommandTestCaseExecutionStruct) (
	rawTestInstructionsToBeSentToExecutionWorkers []newTestInstructionToBeSentToExecutionWorkersStruct,
	rawTestInstructionAttributesToBeSentToExecutionWorkers []newTestInstructionAttributeToBeSentToExecutionWorkersStruct,
	err error) {

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	var testInstructionExecutionUuids []string

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
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars = "AND "

		default:
			// When this is not the first then we need to add 'OR' after previous
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars =
				correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars + "OR "
		}

		// Add the WHERE-values
		correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars =
			correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPar

	}

	// *** Process TestInstructions ***
	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT DP.\"DomainUuid\", DP.\"DomainName\", DP.\"ExecutionWorker Address\", " +
		"TIUE.\"TestInstructionExecutionUuid\", TIUE.\"TestInstructionOriginalUuid\", TIUE.\"TestInstructionName\", " +
		"TIUE.\"TestInstructionMajorVersionNumber\", TIUE.\"TestInstructionMinorVersionNumber\", " +
		"TIUE.\"TestDataSetUuid\", TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\" "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" TIUE, " +
		"\"" + usedDBSchema + "\".\"DomainParameters\" DP "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestInstructionExecutionStatus\" = 0 AND "
	sqlToExecute = sqlToExecute + "DP.\"DomainUuid\" = TIUE.\"DomainUuid\" "
	sqlToExecute = sqlToExecute + correctTestCaseExecutionUuidAndTestCaseExecutionVersionPars
	sqlToExecute = sqlToExecute + "ORDER BY DP.\"DomainUuid\" ASC, TIUE.\"TestInstructionExecutionUuid\" ASC "
	sqlToExecute = sqlToExecute + "; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "eb59969b-d76c-46fd-9d54-f6fa53c28113",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadNewTestInstructionToBeSentToExecutionWorkers'")
	}

	// Query DB
	// Execute Query CloudDB
	rows, err := dbTransaction.Query(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "ac262275-5f05-48c8-982a-46ff2392d3f2",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, nil, err
	}

	// Extract data from DB result set
	for rows.Next() {

		var tempTestInstructionAndAttributeData newTestInstructionToBeSentToExecutionWorkersStruct

		err := rows.Scan(

			&tempTestInstructionAndAttributeData.domainUuid,
			&tempTestInstructionAndAttributeData.domainName,
			&tempTestInstructionAndAttributeData.executionWorkerAddress,
			&tempTestInstructionAndAttributeData.testInstructionExecutionUuid,
			&tempTestInstructionAndAttributeData.testInstructionOriginalUuid,
			&tempTestInstructionAndAttributeData.testInstructionName,
			&tempTestInstructionAndAttributeData.testInstructionMajorVersionNumber,
			&tempTestInstructionAndAttributeData.testInstructionMinorVersionNumber,
			&tempTestInstructionAndAttributeData.testDataSetUuid,
			&tempTestInstructionAndAttributeData.TestCaseExecutionUuid,
			&tempTestInstructionAndAttributeData.testCaseExecutionVersion,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "1173f1f3-f9e5-411d-a129-7ee5ca336762",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, nil, err
		}

		// Add Queue-message to slice of messages
		rawTestInstructionsToBeSentToExecutionWorkers = append(rawTestInstructionsToBeSentToExecutionWorkers, tempTestInstructionAndAttributeData)

		// Add TestInstructionExecutionUuid to array to be used in next SQL for Attributes
		testInstructionExecutionUuids = append(testInstructionExecutionUuids, tempTestInstructionAndAttributeData.testInstructionExecutionUuid)
	}

	// If there were no TestInstructions then it can't be any Attributes, so exit
	if len(rawTestInstructionsToBeSentToExecutionWorkers) == 0 {
		return nil, nil, nil
	}

	// *** Process TestInstructionAttributes ***
	sqlToExecute = ""
	sqlToExecute = sqlToExecute + "SELECT TIAUE.\"TestInstructionExecutionUuid\", TIAUE.\"TestInstructionAttributeType\", TIAUE.\"TestInstructionAttributeUuid\", " +
		"TIAUE.\"TestInstructionAttributeName\", TIAUE.\"AttributeValueAsString\", TIAUE.\"AttributeValueUuid\", " +
		"TIAUE.\"TestInstructionAttributeTypeUuid\", TIAUE.\"TestInstructionAttributeTypeName\" "

	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionAttributesUnderExecution\" TIAUE "
	sqlToExecute = sqlToExecute + "WHERE TIAUE.\"TestInstructionExecutionUuid\" IN " + common_config.GenerateSQLINArray(testInstructionExecutionUuids)

	sqlToExecute = sqlToExecute + "ORDER BY TIAUE.\"TestInstructionExecutionUuid\" ASC, TIAUE.\"TestInstructionAttributeUuid\" ASC "
	sqlToExecute = sqlToExecute + "; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "92f9ceec-1fb1-45e8-9166-091d2762a13c",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadNewTestInstructionToBeSentToExecutionWorkers'")
	}

	// Query DB
	// Execute Query CloudDB
	rows, err = dbTransaction.Query(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "ea92876d-7aa0-4052-afd3-60dcfc79e32c",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return nil, nil, err
	}

	// Extract data from DB result set
	for rows.Next() {

		var tempTestInstructionAttribute newTestInstructionAttributeToBeSentToExecutionWorkersStruct

		err := rows.Scan(

			&tempTestInstructionAttribute.testInstructionExecutionUuid,
			&tempTestInstructionAttribute.testInstructionAttributeType,
			&tempTestInstructionAttribute.testInstructionAttributeUuid,
			&tempTestInstructionAttribute.testInstructionAttributeName,
			&tempTestInstructionAttribute.attributeValueAsString,
			&tempTestInstructionAttribute.attributeValueUuid,
			&tempTestInstructionAttribute.testInstructionExecutionTypeUuid,
			&tempTestInstructionAttribute.testInstructionExecutionTypeName,
		)

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "bb073f45-0207-475d-ae8f-255bb8ee0cf6",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return nil, nil, err
		}

		// Add Queue-message to slice of messages
		rawTestInstructionAttributesToBeSentToExecutionWorkers = append(rawTestInstructionAttributesToBeSentToExecutionWorkers, tempTestInstructionAttribute)

	}

	return rawTestInstructionsToBeSentToExecutionWorkers, rawTestInstructionAttributesToBeSentToExecutionWorkers, err

}

// Holds address, to the execution worker, and a message that will be sent to worker
type processTestInstructionExecutionRequestAndResponseMessageContainer struct {
	domainUuid                              string
	addressToExecutionWorker                string
	testCaseExecutionUuid                   string
	testCaseExecutionVersion                int
	processTestInstructionExecutionRequest  *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest
	processTestInstructionExecutionResponse *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionResponse
}

// Transform Raw TestInstructions from DB into messages ready to be sent over gRPC to Execution Workers
func (executionEngine *TestInstructionExecutionEngineStruct) transformRawTestInstructionsAndAttributeIntoGrpcMessages(rawTestInstructionsToBeSentToExecutionWorkers []newTestInstructionToBeSentToExecutionWorkersStruct, rawTestInstructionAttributesToBeSentToExecutionWorkers []newTestInstructionAttributeToBeSentToExecutionWorkersStruct) (testInstructionsToBeSentToExecutionWorkers []*processTestInstructionExecutionRequestAndResponseMessageContainer, err error) {

	attributesMap := make(map[string]*[]newTestInstructionAttributeToBeSentToExecutionWorkersStruct)

	// Create map for attributes to be able to find attributes in an easier way
	for _, rawTestInstructionAttribute := range rawTestInstructionAttributesToBeSentToExecutionWorkers {
		attributesSliceReference, existsInMap := attributesMap[rawTestInstructionAttribute.testInstructionExecutionUuid]

		if existsInMap == true {
			// Exists, so just append value to slice
			*attributesSliceReference = append(*attributesSliceReference, rawTestInstructionAttribute)
		} else {
			// Doesn't exist, so create new slice and add reference to Map
			var newAttributesSlice []newTestInstructionAttributeToBeSentToExecutionWorkersStruct
			newAttributesSlice = append(newAttributesSlice, rawTestInstructionAttribute)
			attributesMap[rawTestInstructionAttribute.testInstructionExecutionUuid] = &newAttributesSlice
		}
	}

	// Loop TestInstruction build message ready to be sent over gRPC to workers
	for _, rawTestInstructionData := range rawTestInstructionsToBeSentToExecutionWorkers {

		var attributesForTestInstruction []*fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionAttributeMessage

		// Extract all attributes that belongs to a certain TestInstructionExecution
		attributesSlice, existsInMap := attributesMap[rawTestInstructionData.testInstructionExecutionUuid]
		if existsInMap == false || len(*attributesSlice) == 0 {
			// No attributes for the TestInstruction
			attributesForTestInstruction = []*fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionAttributeMessage{}
		} else {
			// Create Attributes-message to be added to TestInstructionExecution-message

			var newProcessTestInstructionExecutionRequest_TestInstructionAttributeMessage *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionAttributeMessage

			for _, attributeInSlice := range *attributesSlice {

				newProcessTestInstructionExecutionRequest_TestInstructionAttributeMessage = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionAttributeMessage{
					TestInstructionAttributeType: fenixExecutionWorkerGrpcApi.TestInstructionAttributeTypeEnum(attributeInSlice.testInstructionAttributeType),
					TestInstructionAttributeUuid: attributeInSlice.testInstructionAttributeUuid,
					TestInstructionAttributeName: attributeInSlice.testInstructionAttributeName,
					AttributeValueAsString:       attributeInSlice.attributeValueAsString,
					AttributeValueUuid:           attributeInSlice.attributeValueUuid,
				}

				// Append to TestInstructionsAttributes-message
				attributesForTestInstruction = append(attributesForTestInstruction, newProcessTestInstructionExecutionRequest_TestInstructionAttributeMessage)
			}
		}

		// Create The TestInstruction and its attributes object, to be sent late
		var newTestInstructionToBeSentToExecutionWorkers *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionExecutionMessage
		newTestInstructionToBeSentToExecutionWorkers = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionExecutionMessage{
			TestInstructionExecutionUuid: rawTestInstructionData.testInstructionExecutionUuid,
			TestInstructionUuid:          rawTestInstructionData.testInstructionOriginalUuid,
			TestInstructionName:          rawTestInstructionData.testInstructionName,
			MajorVersionNumber:           uint32(rawTestInstructionData.testInstructionMajorVersionNumber),
			MinorVersionNumber:           uint32(rawTestInstructionData.testInstructionMinorVersionNumber),
			TestInstructionAttributes:    attributesForTestInstruction,
		}

		// Create the TestData object, to be sent later
		var newTestDataToBeSentToExecutionWorker *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestDataMessage
		newTestDataToBeSentToExecutionWorker = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestDataMessage{
			TestDataSetUuid:           rawTestInstructionData.testDataSetUuid,
			ManualOverrideForTestData: []*fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestDataMessage_ManualOverrideForTestDataMessage{},
		}

		// Build the full request to later be sent to Worker
		var newProcessTestInstructionExecutionReveredRequest *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest
		newProcessTestInstructionExecutionReveredRequest = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest{
			ProtoFileVersionUsedByClient: 0,
			TestInstruction:              newTestInstructionToBeSentToExecutionWorkers,
			TestData:                     newTestDataToBeSentToExecutionWorker,
		}

		// Create one full TestInstruction-coontainer, including address to Worker, so TestInstruction can be sent to Execution Worker over gPRC
		var newProcessTestInstructionExecutionRequestMessageContainer processTestInstructionExecutionRequestAndResponseMessageContainer
		newProcessTestInstructionExecutionRequestMessageContainer = processTestInstructionExecutionRequestAndResponseMessageContainer{
			addressToExecutionWorker:               rawTestInstructionData.executionWorkerAddress,
			processTestInstructionExecutionRequest: newProcessTestInstructionExecutionReveredRequest,
			domainUuid:                             rawTestInstructionData.domainUuid,
			testCaseExecutionUuid:                  rawTestInstructionData.TestCaseExecutionUuid,
			testCaseExecutionVersion:               rawTestInstructionData.testCaseExecutionVersion,
		}

		// Add the TestInstruction-container to slice of containers
		testInstructionsToBeSentToExecutionWorkers = append(testInstructionsToBeSentToExecutionWorkers, &newProcessTestInstructionExecutionRequestMessageContainer)

	}

	return testInstructionsToBeSentToExecutionWorkers, err
}

// Transform Raw TestInstructions from DB into messages ready to be sent over gRPC to Execution Workers
func (executionEngine *TestInstructionExecutionEngineStruct) sendTestInstructionExecutionsToWorker(
	testInstructionsToBeSentToExecutionWorkers []*processTestInstructionExecutionRequestAndResponseMessageContainer) (err error) {

	executionEngine.logger.WithFields(logrus.Fields{
		"id": "79164f56-efe1-4700-910d-4f6783e305bd",
		"testInstructionsToBeSentToExecutionWorkers": testInstructionsToBeSentToExecutionWorkers,
	}).Debug("Incoming 'sendTestInstructionExecutionsToWorker'")

	defer executionEngine.logger.WithFields(logrus.Fields{
		"id": "c54008e1-2ec9-491a-946a-cacd2ddacdbd",
	}).Debug("Outgoing 'sendTestInstructionExecutionsToWorker'")

	// If there are nothing to send then just exit
	if len(testInstructionsToBeSentToExecutionWorkers) == 0 {
		return nil
	}

	// Set up instance to use for execution gPRC
	var fenixExecutionWorkerObject *messagesToExecutionWorker.MessagesToExecutionWorkerServerObjectStruct
	fenixExecutionWorkerObject = &messagesToExecutionWorker.MessagesToExecutionWorkerServerObjectStruct{Logger: executionEngine.logger}

	// Loop all TestInstructionExecutions and send them to correct Worker for executions
	for _, testInstructionToBeSentToExecutionWorkers := range testInstructionsToBeSentToExecutionWorkers {

		// Allocate TimeOut-timer
		// Create a message with TestInstructionExecution to be sent to TimeOutEngine ta be able to allocate a TimeOutTimer
		var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
		tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
			TestInstructionExecutionUuid:    testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid,
			TestInstructionExecutionVersion: 1,
		}

		var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
		tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
			TimeOutChannelCommand:                   common_config.TimeOutChannelCommandAllocateTestInstructionExecutionToTimeOutTimer,
			TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutions,
			SendID:                                  "6e7185e2-8b59-4bf0-aebb-96ab290a19ef",
		}

		// Calculate Execution Track
		var executionTrack int
		executionTrack = common_config.CalculateExecutionTrackNumber(
			testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid)

		// Send message on TimeOutEngineChannel to Add TestInstructionExecution to Timer-queue
		*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrack] <- tempTimeOutChannelCommand

		responseFromWorker := fenixExecutionWorkerObject.SendProcessTestInstructionExecutionToExecutionWorkerServer(
			testInstructionToBeSentToExecutionWorkers.domainUuid,
			testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest)

		executionEngine.logger.WithFields(logrus.Fields{
			"id":                 "7a725e82-d3f4-4cb6-9910-099d8d6dc14e",
			"responseFromWorker": responseFromWorker,
			"TimeOut time":       responseFromWorker.ExpectedExecutionDuration.String(),
		}).Debug("Response from Worker when sending TestInstructionExecution to Worker")

		testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionResponse = responseFromWorker

		//fmt.Println(responseFromWorker)
	}
	return err
}

// Update status on TestInstructions that could be sent to workers
func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestInstructionsExecutionInCloudDB(dbTransaction pgx.Tx, testInstructionsToBeSentToExecutionWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer) (err error) {

	// If there are nothing to update then just exit
	if len(testInstructionsToBeSentToExecutionWorkersAndTheResponse) == 0 {
		return nil
	}

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	// Create Update Statement for response regarding TestInstructionExecution
	for _, testInstructionExecutionResponse := range testInstructionsToBeSentToExecutionWorkersAndTheResponse {

		var tempTestInstructionExecutionStatus string
		var numberOfResend int

		// Save information about TestInstructionExecution when we got a positive response from Worker
		if testInstructionExecutionResponse.processTestInstructionExecutionResponse.AckNackResponse.AckNack == true {
			tempTestInstructionExecutionStatus = "1" // TIE_EXECUTING

		} else {
			// Extract how many times the TestInstructionExecution have been restarted
			sqlToExecute = ""
			sqlToExecute = sqlToExecute + "SELECT TIUE.\"TestInstructionExecutionUuid\", TIUE.\"TestInstructionInstructionExecutionVersion\", " +
				"TIUE.\"ExecutionResendCounter\" "
			sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestInstructionsUnderExecution\" TIUE "
			sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE TIUE.\"TestInstructionExecutionUuid\" = '%s' ", testInstructionExecutionResponse.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid)
			sqlToExecute = sqlToExecute + " AND "
			sqlToExecute = sqlToExecute + "TIUE.\"ExpectedExecutionEndTimeStamp\" IS NULL AND " +
				"TIUE.\"TestInstructionCanBeReExecuted\" = true "
			sqlToExecute = sqlToExecute + "ORDER BY TIUE.\"TestInstructionInstructionExecutionVersion\" DESC "
			sqlToExecute = sqlToExecute + "LIMIT 1"
			sqlToExecute = sqlToExecute + "; "

			// Log SQL to be executed if Environment variable is true
			if common_config.LogAllSQLs == true {
				common_config.Logger.WithFields(logrus.Fields{
					"Id":           "2533b723-05b7-489d-937d-fdfbcfcd97b2",
					"sqlToExecute": sqlToExecute,
				}).Debug("SQL to be executed within 'updateStatusOnTestInstructionsExecutionInCloudDB'")
			}

			// Execute Query CloudDB
			rows, err := dbTransaction.Query(context.Background(), sqlToExecute)

			if err != nil {
				executionEngine.logger.WithFields(logrus.Fields{
					"Id":           "38fe43e8-ac27-4cbe-af05-79e2bd3193cd",
					"Error":        err,
					"sqlToExecute": sqlToExecute,
				}).Error("Something went wrong when executing SQL")

				return err
			}

			// Extract data from DB result set
			for rows.Next() {

				var tempTestInstructionExecutionUuid string
				var tempTestInstructionInstructionExecutionVersion string
				var tempExecutionResendCounter int32

				err := rows.Scan(

					&tempTestInstructionExecutionUuid,
					&tempTestInstructionInstructionExecutionVersion,
					&tempExecutionResendCounter,
				)

				if err != nil {

					executionEngine.logger.WithFields(logrus.Fields{
						"Id":           "27b6a771-22ce-43a7-abcc-d32da0347785",
						"Error":        err,
						"sqlToExecute": sqlToExecute,
					}).Error("Something went wrong when processing result from database")

					return err
				}
			}

			if numberOfResend+1 < common_config.MaxResendToWorkerWhenNoAnswer {

				// Save information about TestInstructionExecution when we got unknown error response from Worker
				tempTestInstructionExecutionStatus = "9" // TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN
				numberOfResend = numberOfResend + 1

			} else {

				// Max number of reruns already achieved, Save information about TestInstructionExecution when we got unknown error response from Worker
				tempTestInstructionExecutionStatus = "8" // TIE_UNEXPECTED_INTERRUPTION
			}
		}

		sqlToExecute = ""
		sqlToExecute = sqlToExecute + "UPDATE \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" "
		sqlToExecute = sqlToExecute + fmt.Sprintf("SET \"ExpectedExecutionEndTimeStamp\" = '%s', ", common_config.ConvertGrpcTimeStampToStringForDB(testInstructionExecutionResponse.processTestInstructionExecutionResponse.ExpectedExecutionDuration))
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionStatus\" = '%s', ", tempTestInstructionExecutionStatus)
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionStatusUpdateTimeStamp\" = '%s' ", currentDataTimeStamp)
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"numberOfResend\" = %d ", numberOfResend)
		sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestInstructionExecutionUuid\" = '%s' ", testInstructionExecutionResponse.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid)

		sqlToExecute = sqlToExecute + "; "

	}

	// If no positive responses the just exit
	if len(sqlToExecute) == 0 {
		return nil
	}

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "f7c27d7b-478a-490e-abcc-419bind3e3dd",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'updateStatusOnTestInstructionsExecutionInCloudDB'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "96b803db-8459-41b6-9420-e24f9965b425",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	executionEngine.logger.WithFields(logrus.Fields{
		"Id":                       "ffa5c358-ba6c-47bd-a828-d9e5d826f913",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// No errors occurred
	return nil

}

// Update status on TestCases that TestInstructions have been sent to workers
func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestCasesExecutionInCloudDB(dbTransaction pgx.Tx, testInstructionsToBeSentToExecutionWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer) (err error) {

	// If there are nothing to update then just exit
	if len(testInstructionsToBeSentToExecutionWorkersAndTheResponse) == 0 {
		return nil
	}

	// Get a common dateTimeStamp to use
	currentDataTimeStamp := fenixSyncShared.GenerateDatetimeTimeStampForDB()

	usedDBSchema := "FenixExecution" // TODO should this env variable be used? fenixSyncShared.GetDBSchemaName()

	sqlToExecute := ""

	// Create Update Statement for response regarding TestInstructionExecution
	for _, testInstructionExecutionResponse := range testInstructionsToBeSentToExecutionWorkersAndTheResponse {

		var tempTestInstructionExecutionStatus string

		// Save information about TestInstructionExecution when we got a positive response from Worker
		if testInstructionExecutionResponse.processTestInstructionExecutionResponse.AckNackResponse.AckNack == true {
			tempTestInstructionExecutionStatus = "1" // TIE_EXECUTING
		} else {
			// Save information about TestInstructionExecution when we got unknown error response from Worker
			tempTestInstructionExecutionStatus = "3" // TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN
		}

		sqlToExecute = sqlToExecute + "UPDATE \"" + usedDBSchema + "\".\"TestCasesUnderExecution\" "
		sqlToExecute = sqlToExecute + fmt.Sprintf("SET \"TestCaseExecutionStatus\" = '%s', ", tempTestInstructionExecutionStatus)
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionStatusUpdateTimeStamp\" = '%s' ", currentDataTimeStamp)
		sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestCaseExecutionUuid\" = '%s' AND ", testInstructionExecutionResponse.testCaseExecutionUuid)
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestCaseExecutionVersion\" = '%s' ", strconv.Itoa(testInstructionExecutionResponse.testCaseExecutionVersion))
		sqlToExecute = sqlToExecute + "; "

	}

	// If no positive responses the just exit
	if len(sqlToExecute) == 0 {
		return nil
	}

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "481128c9-2070-4f56-a821-f93e1a8c79bb",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'updateStatusOnTestCasesExecutionInCloudDB'")
	}

	// Execute Query CloudDB
	comandTag, err := dbTransaction.Exec(context.Background(), sqlToExecute)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "8a1a5099-9383-4c7c-8fc9-58c3521912a4",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return err
	}

	// Log response from CloudDB
	executionEngine.logger.WithFields(logrus.Fields{
		"Id":                       "433ccd0c-a7a7-4245-b346-b8efedefcfd8",
		"comandTag.Insert()":       comandTag.Insert(),
		"comandTag.Delete()":       comandTag.Delete(),
		"comandTag.Select()":       comandTag.Select(),
		"comandTag.Update()":       comandTag.Update(),
		"comandTag.RowsAffected()": comandTag.RowsAffected(),
		"comandTag.String()":       comandTag.String(),
	}).Debug("Return data for SQL executed in database")

	// No errors occurred
	return nil

}

// Set TimeOut-timers for TestInstructionExecutions in TimerOutEngine if we got an AckNack=true as respons
func (executionEngine *TestInstructionExecutionEngineStruct) setTimeOutTimersForTestInstructionExecutions(testInstructionsToBeSentToExecutionWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer) (err error) {

	// If there are nothing to update then just exit
	if len(testInstructionsToBeSentToExecutionWorkersAndTheResponse) == 0 {
		return nil
	}

	// Create TimeOut-message for all TestInstructionExecutions
	for _, testInstructionExecution := range testInstructionsToBeSentToExecutionWorkersAndTheResponse {

		// Process only TestInstructionExecutions if we got an AckNack=true as respons
		if testInstructionExecution.processTestInstructionExecutionResponse.AckNackResponse.AckNack == true {
			// Create a message with TestInstructionExecution to be sent to TimeOutEngine
			var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
			tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
				TestCaseExecutionUuid:                   testInstructionExecution.testCaseExecutionUuid,
				TestCaseExecutionVersion:                int32(testInstructionExecution.testCaseExecutionVersion),
				TestInstructionExecutionUuid:            testInstructionExecution.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid,
				TestInstructionExecutionVersion:         1,
				TestInstructionExecutionCanBeReExecuted: testInstructionExecution.processTestInstructionExecutionResponse.TestInstructionCanBeReExecuted,
				TimeOutTime:                             testInstructionExecution.processTestInstructionExecutionResponse.ExpectedExecutionDuration.AsTime(),
			}

			var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
			tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
				TimeOutChannelCommand:                   common_config.TimeOutChannelCommandAddTestInstructionExecutionToTimeOutTimer,
				TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutions,
				//TimeOutReturnChannelForTimeOutHasOccurred:                           nil,
				//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer: nil,
				SendID: "9d59fc1b-9b11-4adf-b175-1ebbc60eceae",
			}
			// Calculate Execution Track
			var executionTrack int
			executionTrack = common_config.CalculateExecutionTrackNumber(
				testInstructionExecution.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid)

			// Send message on TimeOutEngineChannel to Add TestInstructionExecution to Timer-queue
			*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrack] <- tempTimeOutChannelCommand

		}
	}

	return err
}

// Remove Allocations for TimeOut-timers for TestInstructionExecutions in TimerOutEngine if we got an AckNack=false as respons
func (executionEngine *TestInstructionExecutionEngineStruct) removeTimeOutTimerAllocationsForTestInstructionExecutions(testInstructionsToBeSentToExecutionWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer) (err error) {

	// If there are nothing to update then just exit
	if len(testInstructionsToBeSentToExecutionWorkersAndTheResponse) == 0 {
		return nil
	}

	// Create TimeOut-message for all TestInstructionExecutions
	for _, testInstructionExecution := range testInstructionsToBeSentToExecutionWorkersAndTheResponse {

		// Process only TestInstructionExecutions if we got an AckNack=fasle as respons
		if testInstructionExecution.processTestInstructionExecutionResponse.AckNackResponse.AckNack == false {
			// Create a message with TestInstructionExecution to be sent to TimeOutEngine
			var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
			tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
				TestCaseExecutionUuid:                   testInstructionExecution.testCaseExecutionUuid,
				TestCaseExecutionVersion:                int32(testInstructionExecution.testCaseExecutionVersion),
				TestInstructionExecutionUuid:            testInstructionExecution.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid,
				TestInstructionExecutionVersion:         1,
				TestInstructionExecutionCanBeReExecuted: testInstructionExecution.processTestInstructionExecutionResponse.TestInstructionCanBeReExecuted,
				TimeOutTime:                             testInstructionExecution.processTestInstructionExecutionResponse.ExpectedExecutionDuration.AsTime(),
			}

			var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
			tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
				TimeOutChannelCommand:                   common_config.TimeOutChannelCommandRemoveAllocationForTestInstructionExecutionToTimeOutTimer,
				TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutions,
				//TimeOutReturnChannelForTimeOutHasOccurred:                           nil,
				//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer: nil,
				SendID: "3f5b5990-c7c6-4cba-9136-f90ba7530981",
			}
			// Calculate Execution Track
			var executionTrack int
			executionTrack = common_config.CalculateExecutionTrackNumber(
				testInstructionExecution.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid)

			// Send message on TimeOutEngineChannel to Add TestInstructionExecution to Timer-queue
			*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrack] <- tempTimeOutChannelCommand

		}
	}

	return err
}
