package testInstructionExecutionEngine

import (
	"FenixExecutionServer/broadcastingEngine_ExecutionStatusUpdate"
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/messagesToExecutionWorker"
	"errors"
	"github.com/google/uuid"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixTestCaseBuilderServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixTestCaseBuilderServer/fenixTestCaseBuilderServerGrpcApi/go_grpc_api"
	"google.golang.org/protobuf/types/known/timestamppb"
	"os"
	"sort"
	"strings"
	"time"
	//"FenixExecutionServer/testInstructionTimeOutEngine"
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	fenixExecutionWorkerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionWorkerGrpcApi/go_grpc_api"
	testInstruction_SendTestDataToThisDomain_version_1_0 "github.com/jlambert68/FenixStandardTestInstructionAdmin/TestInstructionsAndTesInstructionContainersAndAllowedUsers/TestInstructions/TestInstruction_SendTestDataToThisDomain/version_1_0"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"strconv"
)

func (executionEngine *TestInstructionExecutionEngineStruct) sendNewTestInstructionsThatIsWaitingToBeSentWorkerCommitOrRoleBackParallelSave(
	executionTrackNumberReference *int,
	dbTransactionReference *pgx.Tx,
	doCommitNotRoleBackReference *bool,
	testInstructionExecutionsReference *[]broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct,
	channelCommandTestCaseExecutionReference *[]ChannelCommandTestCaseExecutionStruct,
	messageShallBeBroadcastedReference *bool) {

	executionTrackNumber := *executionTrackNumberReference
	dbTransaction := *dbTransactionReference
	doCommitNotRoleBack := *doCommitNotRoleBackReference
	testInstructionExecutions := *testInstructionExecutionsReference
	channelCommandTestCaseExecution := *channelCommandTestCaseExecutionReference
	messageShallBeBroadcasted := *messageShallBeBroadcastedReference

	if doCommitNotRoleBack == true {
		dbTransaction.Commit(context.Background())

		// Only broadcast message if it should be broadcasted
		if messageShallBeBroadcasted == true {

			// Create message to be sent to BroadcastEngine
			var broadcastingMessageForExecutions broadcastingEngine_ExecutionStatusUpdate.BroadcastingMessageForExecutionsStruct
			broadcastingMessageForExecutions = broadcastingEngine_ExecutionStatusUpdate.BroadcastingMessageForExecutionsStruct{
				OriginalMessageCreationTimeStamp: strings.Split(time.Now().UTC().String(), " m=")[0],
				TestCaseExecutions:               nil,
				TestInstructionExecutions:        testInstructionExecutions,
			}

			// Send message to BroadcastEngine over channel
			broadcastingEngine_ExecutionStatusUpdate.BroadcastEngineMessageChannel <- broadcastingMessageForExecutions

			defer common_config.Logger.WithFields(logrus.Fields{
				"id":                               "dc384f61-13a6-4f3b-9e1d-345669cf3947",
				"broadcastingMessageForExecutions": broadcastingMessageForExecutions,
			}).Debug("Sent message on Broadcast channel")

		}

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
	var testInstructionExecutions []broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct

	// All TestCaseExecutions to update status on
	var channelCommandTestCaseExecution []ChannelCommandTestCaseExecutionStruct

	// Defines if a message should be broadcasted for signaling a ExecutionStatus change
	var messageShallBeBroadcasted bool

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
	defer executionEngine.sendNewTestInstructionsThatIsWaitingToBeSentWorkerCommitOrRoleBackParallelSave(
		&executionTrackNumber,
		&txn,
		&doCommitNotRoleBack,
		&testInstructionExecutions,
		&channelCommandTestCaseExecution,
		&messageShallBeBroadcasted)

	// Generate a new TestCaseExecutionUuid-UUID
	//testCaseExecutionUuid := uuidGenerator.New().String()

	// Generate TimeStamp
	//placedOnTestExecutionQueueTimeStamp := time.Now()

	// Load all TestInstructions and their attributes to be sent to the Executions Workers over gRPC
	var rawTestInstructionsToBeSentToExecutionWorkers []newTestInstructionToBeSentToExecutionWorkersStruct
	var rawTestInstructionAttributesToBeSentToExecutionWorkers []newTestInstructionAttributeToBeSentToExecutionWorkersStruct
	rawTestInstructionsToBeSentToExecutionWorkers, rawTestInstructionAttributesToBeSentToExecutionWorkers, err =
		executionEngine.loadNewTestInstructionToBeSentToExecutionWorkers(
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
	var testInstructionsToBeSentToExecutionWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer
	testInstructionsToBeSentToExecutionWorkersAndTheResponse, err = executionEngine.
		transformRawTestInstructionsAndAttributeIntoGrpcMessages(
			rawTestInstructionsToBeSentToExecutionWorkers,
			rawTestInstructionAttributesToBeSentToExecutionWorkers) //(txn)

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "75b01b7b-cb47-47da-8fe9-eea85c38f5e3",
			"error": err,
		}).Error("Some problem when transforming raw rawTestInstructionsToBeSentToExecutionWorkers- and Attributes-data into gRPC-messages")

		return
	}

	// If there are no TestInstructions then exit
	if rawTestInstructionsToBeSentToExecutionWorkers == nil {

		return
	}

	// Send TestInstructionExecutions with their attributes to correct Execution Worker
	err = executionEngine.sendTestInstructionExecutionsToWorker(
		txn,
		testInstructionsToBeSentToExecutionWorkersAndTheResponse)
	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "ef815265-3b67-4b59-862f-26d352795c95",
			"error": err,
		}).Error("Got some problem when sending sending TestInstructionExecutions to Worker")

		// Generate new LogPostUuid
		var logPostUuid uuid.UUID
		logPostUuid, err2 := uuid.NewRandom()
		if err2 != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"id": "799d8348-582b-4699-ba97-f4290446fdf2",
			}).Error("Failed to generate UUID")

		}

		// Create the log post row
		var errLogPostsToAdd []*fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage_LogPostMessage
		var errLogPostToAdd *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage_LogPostMessage
		errLogPostToAdd = &fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage_LogPostMessage{
			LogPostUuid:                         logPostUuid.String(),
			LogPostTimeStamp:                    timestamppb.Now(),
			LogPostStatus:                       fenixExecutionServerGrpcApi.LogPostStatusEnum_EXECUTION_ERROR,
			LogPostText:                         err.Error(),
			FoundVersusExpectedValueForVariable: nil,
		}

		errLogPostsToAdd = append(errLogPostsToAdd, errLogPostToAdd)

		// Create "finalTestInstructionExecutionResultMessage" to be used when processing message
		var finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage
		finalTestInstructionExecutionResultMessage = &fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage{
			ClientSystemIdentification: &fenixExecutionServerGrpcApi.ClientSystemIdentificationMessage{
				DomainUuid:                   testInstructionsToBeSentToExecutionWorkersAndTheResponse[0].domainUuid,
				ProtoFileVersionUsedByClient: 0,
			},
			TestInstructionExecutionUuid:           testInstructionsToBeSentToExecutionWorkersAndTheResponse[0].processTestInstructionExecutionRequest.GetTestInstruction().GetTestInstructionExecutionUuid(),
			TestInstructionExecutionVersion:        testInstructionsToBeSentToExecutionWorkersAndTheResponse[0].processTestInstructionExecutionRequest.GetTestInstruction().GetTestInstructionExecutionVersion(),
			MatureTestInstructionUuid:              testInstructionsToBeSentToExecutionWorkersAndTheResponse[0].processTestInstructionExecutionRequest.GetTestInstruction().GetMatureTestInstructionUuid(),
			TestInstructionExecutionStatus:         fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN,
			TestInstructionExecutionStartTimeStamp: timestamppb.Now(),
			TestInstructionExecutionEndTimeStamp:   timestamppb.Now(),
			ResponseVariables:                      nil,
			LogPosts:                               errLogPostsToAdd,
		}

		// Create Message to be sent to TestInstructionExecutionEngine
		channelCommandMessage := ChannelCommandStruct{
			ChannelCommand: ChannelCommandProblemWhenSendingToWorker,
			FinalTestInstructionExecutionResultMessage: finalTestInstructionExecutionResultMessage,
		}

		// Send Message to TestInstructionExecutionEngine via channel
		*executionEngine.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage

		return

	}

	// Update status on TestInstructions that could be sent to workers
	err = executionEngine.updateStatusOnTestInstructionsExecutionInCloudDB(
		txn,
		testInstructionsToBeSentToExecutionWorkersAndTheResponse)

	if err != nil {

		executionEngine.logger.WithFields(logrus.Fields{
			"id":    "bdbc719a-d354-4ebb-9d4c-d002ec55426c",
			"error": err,
		}).Error("Couldn't update TestInstructionExecutionStatus in CloudDB")

		return

	}

	// Update status on TestCases that TestInstructions have been sent to workers
	err = executionEngine.updateStatusOnTestCasesExecutionInCloudDB(
		txn,
		testInstructionsToBeSentToExecutionWorkersAndTheResponse)

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
		var testInstructionExecutionBroadcastMessages []broadcastingEngine_ExecutionStatusUpdate.TestInstructionExecutionBroadcastMessageStruct
		testInstructionExecutionBroadcastMessages, err = executionEngine.loadTestInstructionExecutionDetailsForBroadcastMessage(
			txn,
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

			//Remove temporary for TimeOut-timer because we got an AckNack=false as response
			err = executionEngine.removeTemporaryTimeOutTimerForTestInstructionExecutions(
				testInstructionsToBeSentToExecutionWorkersAndTheResponse)
			if err != nil {

				executionEngine.logger.WithFields(logrus.Fields{
					"id":    "640602b0-04be-4206-987b-4ccc1cb5d46c",
					"error": err,
				}).Error("Problem when sending TestInstructionExecutions to TimeOutEngine")

				return

			}
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

	/*
		// Set TimeOut-timers for TestInstructionExecutions in TimerOutEngine if we got an AckNack=true as respons
		err = executionEngine.setTimeOutTimersForTestInstructionExecutions(
			testInstructionsToBeSentToExecutionWorkersAndTheResponse,
		)
		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"id":    "7d998314-1a9c-40bb-a7f2-9017b26db58f",
				"error": err,
			}).Error("Problem when sending TestInstructionExecutions to TimeOutEngine")

			return

		}

	*/

	// Commit every database change
	doCommitNotRoleBack = true

	return
}

// Hold one new TestInstruction to be sent to Execution Worker
type newTestInstructionToBeSentToExecutionWorkersStruct struct {
	domainUuid                        string
	domainName                        string
	executionDomainUuid               string
	executionDomainName               string
	executionWorkerAddress            string
	testInstructionExecutionUuid      string
	testInstructionExecutionVersion   int
	testInstructionOriginalUuid       string
	matureTestInstructionUuid         string
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
		"TIUE.\"TestDataSetUuid\", TIUE.\"TestCaseExecutionUuid\", TIUE.\"TestCaseExecutionVersion\"," +
		" \"ExecutionDomainUuid\", \"ExecutionDomainName\", \"MatureTestInstructionUuid\", " +
		"\"TestInstructionInstructionExecutionVersion\" "
	sqlToExecute = sqlToExecute + "FROM \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" TIUE, " +
		"\"" + usedDBSchema + "\".\"DomainParameters\" DP "
	sqlToExecute = sqlToExecute + "WHERE TIUE.\"TestInstructionExecutionStatus\" = " +
		strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_INITIATED)) + " AND "
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
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	rows, err := dbTransaction.Query(ctx, sqlToExecute)
	defer rows.Close()

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
			&tempTestInstructionAndAttributeData.executionDomainUuid,
			&tempTestInstructionAndAttributeData.executionDomainName,
			&tempTestInstructionAndAttributeData.matureTestInstructionUuid,
			&tempTestInstructionAndAttributeData.testInstructionExecutionVersion,
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
	defer rows.Close()

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
	domainUuid                                       string
	executionDomainUuid                              string
	addressToExecutionWorker                         string
	testCaseExecutionUuid                            string
	testCaseExecutionVersion                         int
	processTestInstructionExecutionRequest           *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest
	processTestInstructionExecutionResponse          *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionResponse
	messageInitiatedFromPubSubSend                   bool
	messageInitiatedTestInstructionExecutionResponse bool
}

// Transform Raw TestInstructions from DB into messages ready to be sent over gRPC to Execution Workers
func (executionEngine *TestInstructionExecutionEngineStruct) transformRawTestInstructionsAndAttributeIntoGrpcMessages(
	rawTestInstructionsToBeSentToExecutionWorkers []newTestInstructionToBeSentToExecutionWorkersStruct,
	rawTestInstructionAttributesToBeSentToExecutionWorkers []newTestInstructionAttributeToBeSentToExecutionWorkersStruct) (
	testInstructionsToBeSentToExecutionWorkers []*processTestInstructionExecutionRequestAndResponseMessageContainer,
	err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"id": "c3a06561-7fe7-48e4-ae88-b5afbc3d286b",
		"rawTestInstructionsToBeSentToExecutionWorkers":          rawTestInstructionsToBeSentToExecutionWorkers,
		"rawTestInstructionAttributesToBeSentToExecutionWorkers": rawTestInstructionAttributesToBeSentToExecutionWorkers,
	}).Debug("Incoming 'transformRawTestInstructionsAndAttributeIntoGrpcMessages'")

	defer common_config.Logger.WithFields(logrus.Fields{
		"id": "13dec767-9731-4299-a195-dfbcb59ad1e5",
	}).Debug("Outgoing 'prepareTestInstructionIsNotHandledByThisExecutionInstanceSaveFinalTestInstructionExecutionResultToCloudDB'")

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

				newProcessTestInstructionExecutionRequest_TestInstructionAttributeMessage =
					&fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionReveredRequest_TestInstructionAttributeMessage{
						TestInstructionAttributeType:     fenixExecutionWorkerGrpcApi.TestInstructionAttributeTypeEnum(int32(attributeInSlice.testInstructionAttributeType)),
						TestInstructionAttributeUuid:     attributeInSlice.testInstructionAttributeUuid,
						TestInstructionAttributeName:     attributeInSlice.testInstructionAttributeName,
						AttributeValueAsString:           attributeInSlice.attributeValueAsString,
						AttributeValueUuid:               attributeInSlice.attributeValueUuid,
						TestInstructionAttributeTypeUuid: attributeInSlice.testInstructionExecutionTypeUuid,
						TestInstructionAttributeTypeName: attributeInSlice.testInstructionExecutionTypeName,
					}

				// Append to TestInstructionsAttributes-message
				attributesForTestInstruction = append(attributesForTestInstruction,
					newProcessTestInstructionExecutionRequest_TestInstructionAttributeMessage)
			}
		}

		// Create The TestInstruction and its attributes object, to be sent late
		var newTestInstructionToBeSentToExecutionWorkers *fenixExecutionWorkerGrpcApi.
			ProcessTestInstructionExecutionReveredRequest_TestInstructionExecutionMessage
		newTestInstructionToBeSentToExecutionWorkers = &fenixExecutionWorkerGrpcApi.
			ProcessTestInstructionExecutionReveredRequest_TestInstructionExecutionMessage{
			TestInstructionExecutionUuid:    rawTestInstructionData.testInstructionExecutionUuid,
			TestInstructionOriginalUuid:     rawTestInstructionData.testInstructionOriginalUuid,
			TestInstructionExecutionVersion: int32(rawTestInstructionData.testInstructionExecutionVersion),
			MatureTestInstructionUuid:       rawTestInstructionData.matureTestInstructionUuid,
			TestInstructionName:             rawTestInstructionData.testInstructionName,
			MajorVersionNumber:              uint32(rawTestInstructionData.testInstructionMajorVersionNumber),
			MinorVersionNumber:              uint32(rawTestInstructionData.testInstructionMinorVersionNumber),
			TestInstructionAttributes:       attributesForTestInstruction,
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

		// Create one full TestInstruction-container, including address to Worker, so TestInstruction can be sent to Execution Worker over gPRC
		var newProcessTestInstructionExecutionRequestMessageContainer processTestInstructionExecutionRequestAndResponseMessageContainer
		newProcessTestInstructionExecutionRequestMessageContainer = processTestInstructionExecutionRequestAndResponseMessageContainer{
			addressToExecutionWorker:               rawTestInstructionData.executionWorkerAddress,
			processTestInstructionExecutionRequest: newProcessTestInstructionExecutionReveredRequest,
			domainUuid:                             rawTestInstructionData.domainUuid,
			executionDomainUuid:                    rawTestInstructionData.executionDomainUuid,
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
	dbTransaction pgx.Tx,
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

	// Refactoring  - removed for now
	/*

		// Set a Temporary TimeOut-timer for all TestInstructions
		err = executionEngine.setTemporaryTimeOutTimersForTestInstructionExecutions(testInstructionsToBeSentToExecutionWorkers)
		if err != nil {

			common_config.Logger.WithFields(logrus.Fields{
				"Id": "4db137ae-a1df-4248-af61-88b8b396f74e",
				"testInstructionsToBeSentToExecutionWorkers": testInstructionsToBeSentToExecutionWorkers,
				"err": err.Error(),
			}).Error("Got some error when creating Temporary TimeOut-times")

			return err
		}

	*/

	// Loop all TestInstructionExecutions and send them to correct Worker for executions
	for _, testInstructionToBeSentToExecutionWorkers := range testInstructionsToBeSentToExecutionWorkers {

		var responseFromWorker *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionResponse

		// Depending on if Worker use PubSub to forward message to Connector or not
		if common_config.WorkerIsUsingPubSubWhenSendingTestInstructionExecutions == true {
			// Worker is using PubSub to forward message to Connector

			//Convert outgoing message to Worker to fit PubSub-process
			var processTestInstructionExecutionPubSubRequest *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionPubSubRequest
			processTestInstructionExecutionPubSubRequest = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionPubSubRequest{
				TestCaseExecutionUuid: testInstructionToBeSentToExecutionWorkers.testCaseExecutionUuid,
				DomainIdentificationAnfProtoFileVersionUsedByClient: &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionPubSubRequest_ClientSystemIdentificationMessage{
					DomainUuid:          testInstructionToBeSentToExecutionWorkers.domainUuid,
					ExecutionDomainUuid: testInstructionToBeSentToExecutionWorkers.executionDomainUuid,
					ProtoFileVersionUsedByClient: fenixExecutionWorkerGrpcApi.
						ProcessTestInstructionExecutionPubSubRequest_CurrentFenixExecutionWorkerProtoFileVersionEnum(
							testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.
								ProtoFileVersionUsedByClient),
				},
				TestInstruction: &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionPubSubRequest_TestInstructionExecutionMessage{
					TestInstructionExecutionUuid:    testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid,
					TestInstructionExecutionVersion: uint32(testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionVersion),
					TestInstructionOriginalUuid:     testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestInstruction.TestInstructionOriginalUuid,
					MatureTestInstructionUuid:       testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestInstruction.MatureTestInstructionUuid,
					TestInstructionName:             testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestInstruction.TestInstructionName,
					MajorVersionNumber:              testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestInstruction.MajorVersionNumber,
					MinorVersionNumber:              testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestInstruction.MinorVersionNumber,
					TestInstructionAttributes:       nil, // Is set below
				},
				TestData: &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionPubSubRequest_TestDataMessage{
					TestDataSetUuid:           testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestData.TestDataSetUuid,
					ManualOverrideForTestData: nil, // Is set below
				},
			}

			// Process 'TestInstructionAttributes'
			var tempTestInstructionAttributes []*fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionPubSubRequest_TestInstructionAttributeMessage
			for _, testInstructionAttributesForReversed := range testInstructionToBeSentToExecutionWorkers.
				processTestInstructionExecutionRequest.TestInstruction.TestInstructionAttributes {

				var tempTestInstructionAttribute *fenixExecutionWorkerGrpcApi.
					ProcessTestInstructionExecutionPubSubRequest_TestInstructionAttributeMessage
				tempTestInstructionAttribute = &fenixExecutionWorkerGrpcApi.
					ProcessTestInstructionExecutionPubSubRequest_TestInstructionAttributeMessage{
					TestInstructionAttributeType: fenixExecutionWorkerGrpcApi.
						ProcessTestInstructionExecutionPubSubRequest_TestInstructionAttributeTypeEnum(
							testInstructionAttributesForReversed.TestInstructionAttributeType),
					TestInstructionAttributeUuid:     testInstructionAttributesForReversed.TestInstructionAttributeUuid,
					TestInstructionAttributeName:     testInstructionAttributesForReversed.TestInstructionAttributeName,
					AttributeValueAsString:           testInstructionAttributesForReversed.AttributeValueAsString,
					AttributeValueUuid:               testInstructionAttributesForReversed.AttributeValueUuid,
					TestInstructionAttributeTypeUuid: testInstructionAttributesForReversed.TestInstructionAttributeTypeUuid,
					TestInstructionAttributeTypeName: testInstructionAttributesForReversed.TestInstructionAttributeTypeName,
				}

				// Switch type of attribute, e.g. TextBox, ComboBox and so on
				switch tempTestInstructionAttribute.GetTestInstructionAttributeType() {
				case fenixExecutionWorkerGrpcApi.
					ProcessTestInstructionExecutionPubSubRequest_TestInstructionAttributeTypeEnum(
						fenixTestCaseBuilderServerGrpcApi.TestInstructionAttributeTypeEnum_TEXTBOX):

					// If this is a TestInstruction, by Fenix, that should send TestData to user decided DomainUuid
					// Check if this attribute used for DomainUuid, then set the value for DomainUuid in the TestInstruction
					if tempTestInstructionAttribute.TestInstructionAttributeUuid == string(testInstruction_SendTestDataToThisDomain_version_1_0.
						TestInstructionAttributeUUID_SendTestDataToThisDomain_SendTestDataToThisDomainTextBox) {

						// Set the value in the Attribute itself
						processTestInstructionExecutionPubSubRequest.DomainIdentificationAnfProtoFileVersionUsedByClient.
							DomainUuid = tempTestInstructionAttribute.GetAttributeValueAsString()

					}

					// If this is a TestInstruction, by Fenix, that should send TestData to user decided ExecutionDomain
					// Check if this attribute used for ExecutionDomainUuid, then set the value for ExecutionDomainUuid in the TestInstruction
					if tempTestInstructionAttribute.TestInstructionAttributeUuid == string(testInstruction_SendTestDataToThisDomain_version_1_0.
						TestInstructionAttributeUUID_SendTestDataToThisDomain_SendTestDataToThisExecutionDomainTextBox) {

						processTestInstructionExecutionPubSubRequest.DomainIdentificationAnfProtoFileVersionUsedByClient.
							ExecutionDomainUuid = tempTestInstructionAttribute.GetAttributeValueAsString()

					}

					// If this is a TestInstruction, by Fenix, that should send TestData to user decided ExecutionDomain
					// Load the TestData from Database to be sent with the TestInstruction
					if tempTestInstructionAttribute.TestInstructionAttributeUuid == string(testInstruction_SendTestDataToThisDomain_version_1_0.
						TestInstructionAttributeUUID_SendTestDataToThisDomain_ChosenTestDataAsJsonString) {

						// Load the TestData from the DataData
						var testDataRowAsJsonStringToBeSentToConnector string
						testDataRowAsJsonStringToBeSentToConnector, err = executionEngine.
							loadTestDataToBeSentWithTestInstruction(
								//dbTransaction,
								testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.
									TestInstruction,
								testInstructionToBeSentToExecutionWorkers)

						if err != nil {
							common_config.Logger.WithFields(logrus.Fields{
								"Id":              "a1bd7b94-418d-4ff5-a590-269916ace40f",
								"TestInstruction": testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestInstruction,
								"err":             err.Error(),
							}).Error("Got some error when reading TestData for TestCase from database")

							return err
						}

						// Add the response value to the attribute
						tempTestInstructionAttribute.AttributeValueAsString = testDataRowAsJsonStringToBeSentToConnector

					}

				case fenixExecutionWorkerGrpcApi.
					ProcessTestInstructionExecutionPubSubRequest_TestInstructionAttributeTypeEnum(
						fenixTestCaseBuilderServerGrpcApi.TestInstructionAttributeTypeEnum_COMBOBOX):

					// Do nothing 'AttributeValueAsString' has the value

				case fenixExecutionWorkerGrpcApi.
					ProcessTestInstructionExecutionPubSubRequest_TestInstructionAttributeTypeEnum(
						fenixTestCaseBuilderServerGrpcApi.TestInstructionAttributeTypeEnum_RESPONSE_VARIABLE_COMBOBOX):
					// Replace value in 'AttributeValueAsString' with values from Database

					var responseValueFromPreviousTestInstructionExecution string
					responseValueFromPreviousTestInstructionExecution, err = executionEngine.
						loadResponseVariableToUseForAttribute(
							dbTransaction,
							testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.
								TestInstruction,
							testInstructionToBeSentToExecutionWorkers)

					if err != nil {
						common_config.Logger.WithFields(logrus.Fields{
							"Id":              "28153566-1fb2-4496-ba3a-cc078dd9fdea",
							"TestInstruction": testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestInstruction,
							"err":             err.Error(),
						}).Error("Got some error when reading response variables from database")

						return err
					}

					// Add the response value to the attribute
					tempTestInstructionAttribute.AttributeValueAsString = responseValueFromPreviousTestInstructionExecution

				default:
					common_config.Logger.WithFields(logrus.Fields{
						"Id":                           "e526648d-beb8-492f-b928-6f59a561e533",
						"TestInstructionAttributeType": tempTestInstructionAttribute.GetTestInstructionAttributeType(),
					}).Fatalln("Unhandled TestInstructionAttributeType. Exiting")
				}

				// Append to slice of attributes
				tempTestInstructionAttributes = append(tempTestInstructionAttributes, tempTestInstructionAttribute)

			}

			processTestInstructionExecutionPubSubRequest.TestInstruction.TestInstructionAttributes = tempTestInstructionAttributes

			// Process 'ManualOverrideForTestData'
			var tempManualOverrideForTestDataMessage []*fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionPubSubRequest_TestDataMessage_ManualOverrideForTestDataMessage
			for _, testInstructionAttributesForReversed := range testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest.TestInstruction.TestInstructionAttributes {

				var tempManualOverrideForTestData *fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionPubSubRequest_TestDataMessage_ManualOverrideForTestDataMessage
				tempManualOverrideForTestData = &fenixExecutionWorkerGrpcApi.ProcessTestInstructionExecutionPubSubRequest_TestDataMessage_ManualOverrideForTestDataMessage{
					TestDataSetAttributeUuid:  testInstructionAttributesForReversed.TestInstructionAttributeUuid,
					TestDataSetAttributeName:  testInstructionAttributesForReversed.TestInstructionAttributeName,
					TestDataSetAttributeValue: testInstructionAttributesForReversed.AttributeValueUuid,
				}

				// Append to slice of attributes
				tempManualOverrideForTestDataMessage = append(tempManualOverrideForTestDataMessage, tempManualOverrideForTestData)
			}

			processTestInstructionExecutionPubSubRequest.TestData.ManualOverrideForTestData = tempManualOverrideForTestDataMessage

			responseFromWorker = fenixExecutionWorkerObject.SendProcessTestInstructionExecutionToExecutionWorkerServerPubSub(
				testInstructionToBeSentToExecutionWorkers.domainUuid,
				processTestInstructionExecutionPubSubRequest)

			testInstructionToBeSentToExecutionWorkers.messageInitiatedFromPubSubSend = true

		} else {
			// Worker is not using PubSub to forward message to Connector
			responseFromWorker = fenixExecutionWorkerObject.SendProcessTestInstructionExecutionToExecutionWorkerServer(
				testInstructionToBeSentToExecutionWorkers.domainUuid,
				testInstructionToBeSentToExecutionWorkers.processTestInstructionExecutionRequest)

			testInstructionToBeSentToExecutionWorkers.messageInitiatedFromPubSubSend = false
		}

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

// Load the response variable that was previously sent back from Connector in another TestInstructionExecution
func (executionEngine *TestInstructionExecutionEngineStruct) loadResponseVariableToUseForAttribute(
	dbTransaction pgx.Tx,
	testInstruction *fenixExecutionWorkerGrpcApi.
		ProcessTestInstructionExecutionReveredRequest_TestInstructionExecutionMessage,
	testInstructionToBeSentToExecutionWorkers *processTestInstructionExecutionRequestAndResponseMessageContainer) (
	responseValueFromPreviousTestInstructionExecution string,
	err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"Id":              "150fd4eb-0b1c-413e-a1b2-b1bb12ad10cc",
		"testInstruction": testInstruction,
		"testInstructionToBeSentToExecutionWorkers":         testInstructionToBeSentToExecutionWorkers,
		"responseValueFromPreviousTestInstructionExecution": responseValueFromPreviousTestInstructionExecution,
	}).Debug("Entering: loadResponseVariableToUseForAttribute()")

	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "1e6c0ab2-98e0-4cde-ae48-ce8a4949e6f1",
		}).Debug("Exiting: loadResponseVariableToUseForAttribute()")
	}()

	// Variables used in SQL
	var tempTestCaseExecutionUuid string
	var tempTestCaseExecutionVersion int
	tempTestCaseExecutionUuid = testInstructionToBeSentToExecutionWorkers.testCaseExecutionUuid
	tempTestCaseExecutionVersion = testInstructionToBeSentToExecutionWorkers.testCaseExecutionVersion

	// Extract MatureTestInstructionUuid, from Attribute, to use in SQL
	var potentialMatureTestInstructionUuidList string
	for _, tempAttribute := range testInstruction.TestInstructionAttributes {

		if tempAttribute.TestInstructionAttributeType == fenixExecutionWorkerGrpcApi.TestInstructionAttributeTypeEnum_RESPONSE_VARIABLE_COMBOBOX {

			if len(potentialMatureTestInstructionUuidList) == 0 {
				// First post
				potentialMatureTestInstructionUuidList = fmt.Sprintf("'%s'", tempAttribute.AttributeValueUuid)

			} else {
				// Not first post
				potentialMatureTestInstructionUuidList = potentialMatureTestInstructionUuidList + fmt.Sprintf(", '%s'", tempAttribute.AttributeValueUuid)
			}
		}
	}

	//AttributeValueUuid = db.MatureTestInstructionUuid AND
	//= db.TestCaseExecutionUuid
	//= db.TestCaseExecutionVersion

	// *** Process TestInstructions ***
	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT rvue.\"ResponseVariableValueAsString\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"ResponseVariablesUnderExecution\" rvue "
	sqlToExecute = sqlToExecute + "WHERE rvue.\"TestCaseExecutionUuid\" = '" + tempTestCaseExecutionUuid + "' AND " +
		"rvue.\"TestCaseExecutionVersion\" = " + strconv.Itoa(tempTestCaseExecutionVersion) + " AND " +
		"rvue.\"MatureTestInstructionUuid\" IN (" + potentialMatureTestInstructionUuidList + ") "
	sqlToExecute = sqlToExecute + "ORDER BY rvue.\"InsertedTimeStamp\" DESC "
	sqlToExecute = sqlToExecute + "LIMIT 1; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "eb59969b-d76c-46fd-9d54-f6fa53c28113",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadNewTestInstructionToBeSentToExecutionWorkers'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	var dbURI string

	var (
		dbUser    = fenixSyncShared.MustGetEnvironmentVariable("DB_USER") // e.g. 'my-db-user'
		dbPwd     = fenixSyncShared.MustGetEnvironmentVariable("DB_PASS") // e.g. 'my-db-password'
		dbTCPHost = fenixSyncShared.MustGetEnvironmentVariable("DB_HOST") // e.g. '127.0.0.1' ('172.17.0.1' if deployed to GAE Flex)
		dbPort    = fenixSyncShared.MustGetEnvironmentVariable("DB_PORT") // e.g. '5432'
		dbName    = fenixSyncShared.MustGetEnvironmentVariable("DB_NAME") // e.g. 'my-database'
		//dbPoolMaxConnections = fenixSyncShared.MustGetEnvironmentVariable("DB_POOL_MAX_CONNECTIONS") // e.g. '10'
	)

	// If the optional DB_HOST environment variable is set, it contains
	// the IP address and port number of a TCP connection pool to be created,
	// such as "127.0.0.1:5432". If DB_HOST is not set, a Unix socket
	// connection pool will be created instead.
	if dbTCPHost != "GCP" {
		dbURI = fmt.Sprintf("host=%s user=%s password=%s port=%s database=%s", dbTCPHost, dbUser, dbPwd, dbPort, dbName)

	} else {

		var dbInstanceConnectionName = fenixSyncShared.MustGetEnvironmentVariable("DB_INSTANCE_CONNECTION_NAME")

		socketDir, isSet := os.LookupEnv("DB_SOCKET_DIR")
		if !isSet {
			socketDir = "/cloudsql"
		}

		dbURI = fmt.Sprintf("user=%s password=%s database=%s host=%s/%s", dbUser, dbPwd, dbName, socketDir, dbInstanceConnectionName)

	}

	// Connect to the database
	conn, err := pgx.Connect(ctx, dbURI)
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":  "5a262705-3d3f-42d5-baec-137ccdef14fa",
			"err": err,
		}).Error("Unable to connect to database")

		return "", err

	} else {

		defer conn.Close(context.Background())
	}

	rows, err := conn.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "02481f29-792a-4baf-bf91-155e13d75517",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return "", err
	}

	// Extract data from DB result set
	var foundRow bool

	// Set the number of retries to find a ResponseVariable. Need more than one due fast execution speed
	var foundRowRetry int
	var foundRowRetries int
	foundRowRetry = 1
	foundRowRetries = 10

	for {
		for rows.Next() {

			err = rows.Scan(
				&responseValueFromPreviousTestInstructionExecution,
			)

			if err != nil {

				executionEngine.logger.WithFields(logrus.Fields{
					"Id":           "78fdc874-d376-4726-a75f-e801e65cd950",
					"Error":        err,
					"sqlToExecute": sqlToExecute,
				}).Error("Something went wrong when processing result from database")

				return "", err
			}

			foundRow = true

		}

		// If there were no ResponseVariable then there some that is wrong(should never happen), so exit
		if foundRow == false {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":                                     "e456ec99-baef-42f4-a0a7-a88b3b696373",
				"Error":                                  err,
				"tempTestCaseExecutionUuid":              tempTestCaseExecutionUuid,
				"tempTestCaseExecutionVersion":           tempTestCaseExecutionVersion,
				"potentialMatureTestInstructionUuidList": potentialMatureTestInstructionUuidList,
				"sqlToExecute":                           sqlToExecute,
				"foundRowRetry":                          foundRowRetry,
			}).Error("Couldn't find any ResponseVariable row in database, should never happen")

			// Check if max number of retries was achieved
			if foundRowRetry == foundRowRetries {
				err = errors.New("couldn't find any ResponseVariable row in database, should never happen")

				return "", err
			}
		}

		if foundRowRetry != foundRowRetries {
			// Sleep for a short while and then continue look for ResponseVariable
			time.Sleep(1000 * time.Millisecond)
			foundRowRetry = foundRowRetry + 1
		} else {

			break
		}

	}

	return responseValueFromPreviousTestInstructionExecution, err

}

// Load the TestData that should be sent to the Connector in a separate TestInstruction
func (executionEngine *TestInstructionExecutionEngineStruct) loadTestDataToBeSentWithTestInstruction(
	//dbTransaction pgx.Tx,
	testInstruction *fenixExecutionWorkerGrpcApi.
		ProcessTestInstructionExecutionReveredRequest_TestInstructionExecutionMessage,
	testInstructionToBeSentToExecutionWorkers *processTestInstructionExecutionRequestAndResponseMessageContainer) (
	testDataForTestCaseExecutionAsJson string,
	err error) {

	common_config.Logger.WithFields(logrus.Fields{
		"Id":              "c45cda82-1eca-45f5-b6c8-a39ffaf26119",
		"testInstruction": testInstruction,
		"testInstructionToBeSentToExecutionWorkers": testInstructionToBeSentToExecutionWorkers,
	}).Debug("Entering: loadTestDataToBeSentWithTestInstruction()")

	defer func() {
		common_config.Logger.WithFields(logrus.Fields{
			"Id": "523f4243-905d-4d60-9c42-556e8e42601c",
		}).Debug("Exiting: loadTestDataToBeSentWithTestInstruction()")
	}()

	// Variables used in SQL
	var tempTestCaseExecutionUuid string
	var tempTestCaseExecutionVersion int
	tempTestCaseExecutionUuid = testInstructionToBeSentToExecutionWorkers.testCaseExecutionUuid
	tempTestCaseExecutionVersion = testInstructionToBeSentToExecutionWorkers.testCaseExecutionVersion

	// *** Process TestData ***
	sqlToExecute := ""
	sqlToExecute = sqlToExecute + "SELECT \"TestDataForTestCaseExecutionAsJsonb\" "
	sqlToExecute = sqlToExecute + "FROM \"FenixExecution\".\"TestDataForTestCaseExecution\" tdftce "
	sqlToExecute = sqlToExecute + "WHERE tdftce.\"TestCaseExecutionUuid\" = '" + tempTestCaseExecutionUuid + "' AND " +
		"tdftce.\"TestCaseExecutionVersion\" = " + strconv.Itoa(tempTestCaseExecutionVersion)
	sqlToExecute = sqlToExecute + "; "

	// Log SQL to be executed if Environment variable is true
	if common_config.LogAllSQLs == true {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":           "b96a74b6-bdd6-40e0-8be1-1fb84f0050ed",
			"sqlToExecute": sqlToExecute,
		}).Debug("SQL to be executed within 'loadTestDataToBeSentWithTestInstruction'")
	}

	// Query DB
	var ctx context.Context
	ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer timeOutCancel()

	var dbURI string

	var (
		dbUser    = fenixSyncShared.MustGetEnvironmentVariable("DB_USER") // e.g. 'my-db-user'
		dbPwd     = fenixSyncShared.MustGetEnvironmentVariable("DB_PASS") // e.g. 'my-db-password'
		dbTCPHost = fenixSyncShared.MustGetEnvironmentVariable("DB_HOST") // e.g. '127.0.0.1' ('172.17.0.1' if deployed to GAE Flex)
		dbPort    = fenixSyncShared.MustGetEnvironmentVariable("DB_PORT") // e.g. '5432'
		dbName    = fenixSyncShared.MustGetEnvironmentVariable("DB_NAME") // e.g. 'my-database'
		//dbPoolMaxConnections = fenixSyncShared.MustGetEnvironmentVariable("DB_POOL_MAX_CONNECTIONS") // e.g. '10'
	)

	// If the optional DB_HOST environment variable is set, it contains
	// the IP address and port number of a TCP connection pool to be created,
	// such as "127.0.0.1:5432". If DB_HOST is not set, a Unix socket
	// connection pool will be created instead.
	if dbTCPHost != "GCP" {
		dbURI = fmt.Sprintf("host=%s user=%s password=%s port=%s database=%s", dbTCPHost, dbUser, dbPwd, dbPort, dbName)

	} else {

		var dbInstanceConnectionName = fenixSyncShared.MustGetEnvironmentVariable("DB_INSTANCE_CONNECTION_NAME")

		socketDir, isSet := os.LookupEnv("DB_SOCKET_DIR")
		if !isSet {
			socketDir = "/cloudsql"
		}

		dbURI = fmt.Sprintf("user=%s password=%s database=%s host=%s/%s", dbUser, dbPwd, dbName, socketDir, dbInstanceConnectionName)

	}

	// Connect to the database
	conn, err := pgx.Connect(ctx, dbURI)
	if err != nil {
		common_config.Logger.WithFields(logrus.Fields{
			"Id":  "dd2d9a41-895d-40e7-8038-4e23e5821243",
			"err": err,
		}).Error("Unable to connect to database")

		return "", err

	} else {

		defer conn.Close(context.Background())
	}

	rows, err := conn.Query(ctx, sqlToExecute)
	defer rows.Close()

	if err != nil {
		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "faf4f03f-78a0-47b3-9239-5f0519d3b82c",
			"Error":        err,
			"sqlToExecute": sqlToExecute,
		}).Error("Something went wrong when executing SQL")

		return "", err
	}

	var rowCounter int

	for rows.Next() {

		err = rows.Scan(
			&testDataForTestCaseExecutionAsJson,
		)

		rowCounter = rowCounter + 1

		if err != nil {

			executionEngine.logger.WithFields(logrus.Fields{
				"Id":           "78fdc874-d376-4726-a75f-e801e65cd950",
				"Error":        err,
				"sqlToExecute": sqlToExecute,
			}).Error("Something went wrong when processing result from database")

			return "", err
		}
	}

	// There can't be more than 1 row
	if rowCounter > 1 {

		err = errors.New("more then one row was found in database, regarding TestData for a TestCase")

		executionEngine.logger.WithFields(logrus.Fields{
			"Id":           "b6b1363f-69f4-46a4-b3d5-16d76a660677",
			"rowCounter":   rowCounter,
			"sqlToExecute": sqlToExecute,
			"err":          err,
		}).Error("Something went wrong when processing result from database")

		return "", err
	}

	return testDataForTestCaseExecutionAsJson, err

}

// Update status on TestInstructions that could be sent to workers
func (executionEngine *TestInstructionExecutionEngineStruct) updateStatusOnTestInstructionsExecutionInCloudDB(
	dbTransaction pgx.Tx, testInstructionsToBeSentToExecutionWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer) (err error) {

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
		//TODO - kolla här
		// Save information about TestInstructionExecution when we got a positive response from Worker
		if testInstructionExecutionResponse.processTestInstructionExecutionResponse.AckNackResponse.AckNack == true {
			tempTestInstructionExecutionStatus = strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_EXECUTING)) // "2" // TIE_EXECUTING

		} else {
			// Extract how many times the TestInstructionExecution have been restarted
			sqlToExecute = ""
			sqlToExecute = sqlToExecute + "SELECT TIUE.\"TestInstructionExecutionUuid\", TIUE.\"TestInstructionInstructionExecutionVersion\", " +
				"TIUE.\"TestInstructionExecutionResendCounter\" "
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

			// Query DB
			var ctx context.Context
			ctx, timeOutCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer timeOutCancel()

			rows, err := dbTransaction.Query(ctx, sqlToExecute)
			defer rows.Close()

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
				var tempTestInstructionInstructionExecutionVersion int32
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
				tempTestInstructionExecutionStatus = strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_INITIATED))
				numberOfResend = numberOfResend + 1

			} else {

				// Max number of reruns already achieved, Save information about TestInstructionExecution when we got unknown error response from Worker
				tempTestInstructionExecutionStatus = strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN)) // "9" // TIE_UNEXPECTED_INTERRUPTION
			}
		}

		sqlToExecute = ""
		sqlToExecute = sqlToExecute + "UPDATE \"" + usedDBSchema + "\".\"TestInstructionsUnderExecution\" "
		sqlToExecute = sqlToExecute + fmt.Sprintf("SET \"ExpectedExecutionEndTimeStamp\" = '%s', ", common_config.ConvertGrpcTimeStampToStringForDB(testInstructionExecutionResponse.processTestInstructionExecutionResponse.ExpectedExecutionDuration))
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionStatus\" = '%s', ", tempTestInstructionExecutionStatus)
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"ExecutionStatusUpdateTimeStamp\" = '%s', ", currentDataTimeStamp)
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionResendCounter\" = %d ", numberOfResend)
		sqlToExecute = sqlToExecute + fmt.Sprintf("WHERE \"TestInstructionExecutionUuid\" = '%s' AND ", testInstructionExecutionResponse.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid)
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionStatus\" = %s ",
			strconv.Itoa(int(fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum_TIE_INITIATED)))
		sqlToExecute = sqlToExecute + "AND "
		sqlToExecute = sqlToExecute + fmt.Sprintf("\"TestInstructionExecutionResendCounter\" < %s ",
			strconv.Itoa(common_config.MaxResendToWorkerWhenNoAnswer))
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
			tempTestInstructionExecutionStatus = strconv.Itoa(int(fenixExecutionServerGrpcApi.
				TestInstructionExecutionStatusEnum_TIE_EXECUTING)) //"2" // TIE_EXECUTING
		} else {
			// Save information about TestInstructionExecution when we got unknown error response from Worker
			tempTestInstructionExecutionStatus = strconv.Itoa(int(fenixExecutionServerGrpcApi.
				TestInstructionExecutionStatusEnum_TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN)) // "10" // TIE_UNEXPECTED_INTERRUPTION_CAN_BE_RERUN
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
func (executionEngine *TestInstructionExecutionEngineStruct) setTimeOutTimersForTestInstructionExecutions(
	testInstructionsToBeSentToExecutionWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer) (
	err error) {

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
				TestInstructionExecutionVersion:         testInstructionExecution.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionVersion,
				TestInstructionExecutionCanBeReExecuted: testInstructionExecution.processTestInstructionExecutionResponse.TestInstructionCanBeReExecuted,
				TimeOutTime:                             testInstructionExecution.processTestInstructionExecutionResponse.ExpectedExecutionDuration.AsTime(),
			}

			var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
			tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
				TimeOutChannelCommand:                   common_config.TimeOutChannelCommandAddTestInstructionExecutionToTimeOutTimer,
				TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutions,
				//TimeOutReturnChannelForTimeOutHasOccurred:                           nil,
				//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer: nil,
				SendID:                         "be7a3abb-4eef-41b4-99be-80fc093bbee3",
				MessageInitiatedFromPubSubSend: testInstructionExecution.messageInitiatedFromPubSubSend,
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

// Set Temporary TimeOut-timers for TestInstructionExecutions in TimerOutEngine if we got an AckNack=true as response
// This is done before the Connector responded back with actual TimeOutTime
func (executionEngine *TestInstructionExecutionEngineStruct) setTemporaryTimeOutTimersForTestInstructionExecutions(
	testInstructionsToBeSentToExecutionWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer) (
	err error) {

	// If there are nothing to update then just exit
	if len(testInstructionsToBeSentToExecutionWorkersAndTheResponse) == 0 {
		return nil
	}

	// Create TimeOut-message for all TestInstructionExecutions
	for _, testInstructionExecution := range testInstructionsToBeSentToExecutionWorkersAndTheResponse {

		// Create a message with TestInstructionExecution to be sent to TimeOutEngine
		var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
		tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
			TestCaseExecutionUuid:    testInstructionExecution.testCaseExecutionUuid,
			TestCaseExecutionVersion: int32(testInstructionExecution.testCaseExecutionVersion),
			TestInstructionExecutionUuid: testInstructionExecution.processTestInstructionExecutionRequest.
				TestInstruction.TestInstructionExecutionUuid,
			TestInstructionExecutionVersion: testInstructionExecution.processTestInstructionExecutionRequest.
				TestInstruction.TestInstructionExecutionVersion,
			TestInstructionExecutionCanBeReExecuted: true,
			TimeOutTime:                             time.Now().Add(60 * time.Second), // Hard code 60 second timeout time
		}

		var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
		tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
			TimeOutChannelCommand:                   common_config.TimeOutChannelCommandAddTemporaryTimeOutTimerBeforeCallToWorker,
			TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutions,
			//TimeOutReturnChannelForTimeOutHasOccurred:                           nil,
			//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer: nil,
			SendID:                         "fba445be-800d-4a93-895c-b57a956e76cc",
			MessageInitiatedFromPubSubSend: testInstructionExecution.messageInitiatedFromPubSubSend,
		}
		// Calculate Execution Track
		var executionTrack int
		executionTrack = common_config.CalculateExecutionTrackNumber(
			testInstructionExecution.processTestInstructionExecutionRequest.TestInstruction.TestInstructionExecutionUuid)

		// Send message on TimeOutEngineChannel to Add TestInstructionExecution to Timer-queue
		*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrack] <- tempTimeOutChannelCommand

	}

	return err
}

/*
// Remove Allocations for TimeOut-timers for TestInstructionExecutions in TimerOutEngine if we got an AckNack=false as response
func (executionEngine *TestInstructionExecutionEngineStruct) removeTimeOutTimerAllocationsForTestInstructionExecutions(
	testInstructionsToBeSentToExecutionWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer) (err error) {

	// If there are nothing to update then just exit
	if len(testInstructionsToBeSentToExecutionWorkersAndTheResponse) == 0 {
		return nil
	}

	// Create TimeOut-message for all TestInstructionExecutions
	for _, testInstructionExecution := range testInstructionsToBeSentToExecutionWorkersAndTheResponse {

		// Process only TestInstructionExecutions if we got an AckNack=false as response
		if testInstructionExecution.processTestInstructionExecutionResponse.AckNackResponse.AckNack == false {
			// Create a message with TestInstructionExecution to be sent to TimeOutEngine
			var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
			tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
				TestCaseExecutionUuid:    testInstructionExecution.testCaseExecutionUuid,
				TestCaseExecutionVersion: int32(testInstructionExecution.testCaseExecutionVersion),
				TestInstructionExecutionUuid: testInstructionExecution.processTestInstructionExecutionRequest.
					TestInstruction.TestInstructionExecutionUuid,
				TestInstructionExecutionVersion: testInstructionExecution.processTestInstructionExecutionRequest.
					TestInstruction.TestInstructionExecutionVersion,
				TestInstructionExecutionCanBeReExecuted: testInstructionExecution.processTestInstructionExecutionResponse.
					TestInstructionCanBeReExecuted,
				TimeOutTime: testInstructionExecution.processTestInstructionExecutionResponse.
					ExpectedExecutionDuration.AsTime(),
			}

			var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
			tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
				TimeOutChannelCommand: common_config.
					TimeOutChannelCommandRemoveAllocationForTestInstructionExecutionToTimeOutTimer,
				TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutions,
				//TimeOutReturnChannelForTimeOutHasOccurred:                           nil,
				//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer: nil,
				SendID:                         "3f5b5990-c7c6-4cba-9136-f90ba7530981",
				MessageInitiatedFromPubSubSend: false,
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
*/

// Remove temporary TimeOut-timers for TestInstructionExecutions in TimerOutEngine if we got an AckNack=false as response
func (executionEngine *TestInstructionExecutionEngineStruct) removeTemporaryTimeOutTimerForTestInstructionExecutions(
	testInstructionsToBeSentToExecutionWorkersAndTheResponse []*processTestInstructionExecutionRequestAndResponseMessageContainer) (err error) {

	// If there are nothing to update then just exit
	if len(testInstructionsToBeSentToExecutionWorkersAndTheResponse) == 0 {
		return nil
	}

	// Create TimeOut-message for all TestInstructionExecutions
	for _, testInstructionExecution := range testInstructionsToBeSentToExecutionWorkersAndTheResponse {

		// Decide which tempTimeOutChannelCommand should be used
		var tempTimeOutChannelCommandType common_config.TimeOutChannelCommandType

		// We are coming from when we tried to Send to Worker
		if testInstructionExecution.processTestInstructionExecutionResponse.AckNackResponse.AckNack == false {
			tempTimeOutChannelCommandType = common_config.
				TimeOutChannelCommandRemoveTemporaryTimeOutTimerDueToProblemInCallToWorker
		}

		// We are coming from when we got response from Connector
		if testInstructionExecution.messageInitiatedTestInstructionExecutionResponse == true {
			tempTimeOutChannelCommandType = common_config.
				TimeOutChannelCommandRemoveTemporaryTimeOutTimerDueToResponseFromConnector
		}

		// Process only TestInstructionExecutions if we got an AckNack=false as response
		if testInstructionExecution.processTestInstructionExecutionResponse.AckNackResponse.AckNack == false ||
			testInstructionExecution.messageInitiatedTestInstructionExecutionResponse == true {
			// Create a message with TestInstructionExecution to be sent to TimeOutEngine
			var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
			tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
				TestCaseExecutionUuid:    testInstructionExecution.testCaseExecutionUuid,
				TestCaseExecutionVersion: int32(testInstructionExecution.testCaseExecutionVersion),
				TestInstructionExecutionUuid: testInstructionExecution.processTestInstructionExecutionRequest.
					TestInstruction.TestInstructionExecutionUuid,
				TestInstructionExecutionVersion: testInstructionExecution.processTestInstructionExecutionRequest.
					TestInstruction.TestInstructionExecutionVersion,
				TestInstructionExecutionCanBeReExecuted: testInstructionExecution.processTestInstructionExecutionResponse.
					TestInstructionCanBeReExecuted,
				TimeOutTime: testInstructionExecution.processTestInstructionExecutionResponse.
					ExpectedExecutionDuration.AsTime(),
			}

			var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
			tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
				TimeOutChannelCommand:                   tempTimeOutChannelCommandType,
				TimeOutChannelTestInstructionExecutions: tempTimeOutChannelTestInstructionExecutions,
				//TimeOutReturnChannelForTimeOutHasOccurred:                           nil,
				//TimeOutReturnChannelForExistsTestInstructionExecutionInTimeOutTimer: nil,
				SendID:                         "4eb5a7d6-ccce-4755-8096-b58db24e0924",
				MessageInitiatedFromPubSubSend: false,
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
