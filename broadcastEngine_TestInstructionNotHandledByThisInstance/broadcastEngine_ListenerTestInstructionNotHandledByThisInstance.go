package broadcastEngine_TestInstructionNotHandledByThisInstance

import (
	"FenixExecutionServer/common_config"
	"FenixExecutionServer/testInstructionExecutionEngine"
	"context"
	"encoding/json"
	"errors"
	fenixExecutionServerGrpcApi "github.com/jlambert68/FenixGrpcApi/FenixExecutionServer/fenixExecutionServerGrpcApi/go_grpc_api"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"log"
	"strconv"
	"time"
)

// InitiateAndStartBroadcastNotifyEngine
// Start listen for Broadcasts regarding change in status TestCaseExecutions and TestInstructionExecutions
func InitiateAndStartBroadcastNotifyEngine() {

	go func() {
		for {
			err := BroadcastListener()
			if err != nil {
				log.Println("unable start listener:", err)

				common_config.Logger.WithFields(logrus.Fields{
					"Id":  "626b47aa-7c37-499f-b4ca-4defacd17433",
					"err": err,
				}).Error("Unable to start Broadcast listener. Will retry in 5 seconds")
			}
			time.Sleep(time.Second * 5)
		}
	}()
}

func BroadcastListener() error {

	var err error
	var broadcastingMessageForExecutions common_config.BroadcastingMessageForTestInstructionExecutionsStruct

	if fenixSyncShared.DbPool == nil {
		return errors.New("empty pool reference")
	}

	conn, err := fenixSyncShared.DbPool.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(context.Background(), "LISTEN testInstructionNotHandledByThisInstance")
	if err != nil {
		return err
	}

	for {
		notification, err := conn.Conn().WaitForNotification(context.Background())
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":  "c03e2c05-1bc9-4d14-a9f9-c79f2c39093e",
				"err": err,
			}).Error("Error waiting for notification")

			// Restart broadcast engine when error occurs. Most probably because nothing is coming
			//defer func() {
			//	_ = BroadcastListener()
			//}()
			return err
		}

		common_config.Logger.WithFields(logrus.Fields{
			"Id":                        "60cfcd19-3466-4c7e-8628-53de45a0d74c",
			"accepted message from pid": notification.PID,
			"channel":                   notification.Channel,
			"payload":                   notification.Payload,
		}).Debug("Got Broadcast message from Postgres Database")

		err = json.Unmarshal([]byte(notification.Payload), &broadcastingMessageForExecutions)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":  "b363640f-502f-4ef5-bdd2-f57b54f824e8",
				"err": err,
			}).Error("Got some error when Unmarshal incoming json over Broadcast system")
		} else {

			// Break down 'broadcastingMessageForExecutions' and send correct content to correct sSubscribers.

			// Refactoring  - removed for now
			//convertToChannelMessageAndPutOnChannels(broadcastingMessageForExecutions)

		}
	}
}

// Break down 'broadcastingMessageForExecutions' and send correct content to correct sSubscribers.
func convertToChannelMessageAndPutOnChannels(broadcastingMessageForExecutions common_config.BroadcastingMessageForTestInstructionExecutionsStruct) {
	/*
		//var originalMessageCreationTimeStamp time.Time
		var err error

		var timeStampLayoutForParser string //:= "2006-01-02 15:04:05.999999999 -0700 MST"

		// Convert Original Message creation Timestamp into time-variable
		timeStampLayoutForParser, err = common_config.GenerateTimeStampParserLayout(broadcastingMessageForExecutions.OriginalMessageCreationTimeStamp)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":  "d2cb561b-9976-407a-a263-a588529019f1",
				"err": err,
				"broadcastingMessageForExecutions.OriginalMessageCreationTimeStamp": broadcastingMessageForExecutions.OriginalMessageCreationTimeStamp,
			}).Error("Couldn't generate parser layout from TimeStamp")

			return
		}

		originalMessageCreationTimeStamp, err = time.Parse(timeStampLayoutForParser, broadcastingMessageForExecutions.OriginalMessageCreationTimeStamp)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                               "422159b0-de42-4b5d-a707-34dfabbf5082",
				"err":                              err,
				"broadcastingMessageForExecutions": broadcastingMessageForExecutions,
			}).Error("Couldn't parse TimeStamp in Broadcast-message")

			return
		}
	*/
	// Convert Original message creation Timestamp into gRPC-version
	//var originalMessageCreationTimeStampForGrpc *timestamppb.Timestamp
	//originalMessageCreationTimeStampForGrpc = timestamppb.New(originalMessageCreationTimeStamp)

	// Loop all TestInstructionExecutions (should only be one in normal case)
	for _, tempTestInstructionExecution := range broadcastingMessageForExecutions.TestInstructionExecutions {

		// Define Execution Track based on "lowest "TestCaseExecutionUuid
		var executionTrackNumber int
		executionTrackNumber = common_config.CalculateExecutionTrackNumber(
			tempTestInstructionExecution.TestInstructionExecutionUuid)

		// *** Check if the TestInstruction is kept in this ExecutionServer-instance ***

		// Create Response channel from TimeOutEngine to get answer if TestInstructionExecution is handled by this instance
		var timeOutResponseChannelForIsThisHandledByThisExecutionInstance common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceType
		timeOutResponseChannelForIsThisHandledByThisExecutionInstance = make(chan common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct)

		// Convert string-version into int32-version
		var tempTestInstructionVersion int
		var err error
		tempTestInstructionVersion, err = strconv.Atoi(tempTestInstructionExecution.TestInstructionExecutionVersion)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":  "0f9b23d2-168f-44dd-a0f2-a6c09b6c262d",
				"err": err,
				"tempTestInstructionExecution.TestInstructionExecutionUuid":                                tempTestInstructionExecution.TestInstructionExecutionUuid,
				"tempTestInstructionExecution.TestInstructionExecutionVersion":                             tempTestInstructionExecution.TestInstructionExecutionVersion,
				"tempTestInstructionExecution.TestInstructionExecutionMessageReceivedByWrongExecutionType": tempTestInstructionExecution.TestInstructionExecutionMessageReceivedByWrongExecutionType,
			}).Error("Couldn't convert 'TestInstructionExecutionVersion' into an integer. Dropping TestInstructionExecution")

			// Drop this message and continue with next message
			continue
		}

		var tempTimeOutChannelTestInstructionExecutions common_config.TimeOutChannelCommandTestInstructionExecutionStruct
		tempTimeOutChannelTestInstructionExecutions = common_config.TimeOutChannelCommandTestInstructionExecutionStruct{
			TestCaseExecutionUuid:                   "",
			TestCaseExecutionVersion:                0,
			TestInstructionExecutionUuid:            tempTestInstructionExecution.TestInstructionExecutionUuid,
			TestInstructionExecutionVersion:         int32(tempTestInstructionVersion),
			TestInstructionExecutionCanBeReExecuted: false,
			TimeOutTime:                             time.Time{},
		}

		var tempTimeOutChannelCommand common_config.TimeOutChannelCommandStruct
		tempTimeOutChannelCommand = common_config.TimeOutChannelCommandStruct{
			TimeOutChannelCommand:                                                   common_config.TimeOutChannelCommandVerifyIfTestInstructionIsHandledByThisExecutionInstance,
			TimeOutChannelTestInstructionExecutions:                                 tempTimeOutChannelTestInstructionExecutions,
			TimeOutReturnChannelForTimeOutHasOccurred:                               nil,
			TimeOutResponseChannelForDurationUntilTimeOutOccurs:                     nil,
			TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstance: &timeOutResponseChannelForIsThisHandledByThisExecutionInstance,
			SendID:                         "4d545fda-d9e4-4d35-b8af-4bbbbacf971e",
			MessageInitiatedFromPubSubSend: false,
		}

		// Send message on TimeOutEngineChannel to get information about if TestInstructionExecution is handled by this Execution-instance
		*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrackNumber] <- tempTimeOutChannelCommand

		// Response from TimeOutEngine
		var timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct

		// Wait for response from TimeOutEngine
		timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue =
			<-timeOutResponseChannelForIsThisHandledByThisExecutionInstance

		// If TestInstructionExecution is not handled by this Execution-instance then wait 'x' seconds and
		// if no one claims the TestInstructionExecution from database-table then claim responsibility of the TestInstructionExecution
		if timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue.
			TestInstructionIsHandledByThisExecutionInstance == false {

			// Sleep to see if someone claims the TestInstructionExecution from database
			time.Sleep(time.Duration(common_config.SleepTimeInSecondsBeforeClaimingTestInstructionExecutionFromDatabase) * time.Second)

			// Check the TestInstructionExecution is claimed by some other ExecutionServer
			var foundInDatabase bool
			foundInDatabase, err = prepareCountTestInstructionExecutionMessagesReceivedByWrongExecutionInstance(
				tempTestInstructionExecution.TestInstructionExecutionUuid,
				int32(tempTestInstructionVersion))

			if err != nil {
				common_config.Logger.WithFields(logrus.Fields{
					"id":                           "3816de0b-f5f9-4f55-9f0d-cb04f4dee0da",
					"TestInstructionExecutionUuid": tempTestInstructionExecution.TestInstructionExecutionUuid,
					"TestInstructionVersion":       tempTestInstructionVersion,
					"error":                        err,
				}).Error("Problem when checking if TestInstructionExecution, received by wrong ExecutionServer, exists in database.")

				// Drop this message and continue with next message
				continue
			}

			// Some other ExecutionServer claimed the TestInstructionExecution
			if foundInDatabase == false {
				continue
			}

			// Claim the TestInstructionExecution
			timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue.
				TestInstructionIsHandledByThisExecutionInstance = true
		}

		// Verify if TestInstructionExecution is handled by this Execution-instance
		if timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue.
			TestInstructionIsHandledByThisExecutionInstance == true {
			// *** TestInstructionExecution is handled by this Execution-instance ***

			// Load TestInstructionExecution from database and then remove data in database
			var messagesReceivedByWrongExecutionInstance []*common_config.TestInstructionExecutionMessageReceivedByWrongExecutionStruct
			messagesReceivedByWrongExecutionInstance, err = prepareLoadTestInstructionExecutionMessagesReceivedByWrongInstance(
				tempTestInstructionExecution.TestInstructionExecutionUuid,
				int32(tempTestInstructionVersion))

			if err != nil {
				common_config.Logger.WithFields(logrus.Fields{
					"Id":  "4437dfb5-a568-4e1b-b1fb-cbc8cf4913ee",
					"err": err,
					"tempTestInstructionExecution.TestInstructionExecutionUuid":    tempTestInstructionExecution.TestInstructionExecutionUuid,
					"tempTestInstructionExecution.TestInstructionExecutionVersion": tempTestInstructionExecution.TestInstructionExecutionVersion,
				}).Error("Problems when reading messages, regarding TestInstructionExecution, received by the wrong Execution-instance")

				// Drop this message and continue with next message
				continue
			}

			// Loop through the messaged and process them
			for _, tempMessageReceivedByWrongExecutionInstance := range messagesReceivedByWrongExecutionInstance {

				// What kind of message is it?
				switch tempMessageReceivedByWrongExecutionInstance.MessageType {

				case common_config.FinalTestInstructionExecutionResultMessageType:
					// Final Execution result for TestInstructionExecution

					// Convert json-strings into byte-arrays
					var tempFinalTestInstructionExecutionResultMessageAsByteArray []byte
					tempFinalTestInstructionExecutionResultMessageAsByteArray = []byte(tempMessageReceivedByWrongExecutionInstance.MessageAsJsonString)

					// Convert json-byte-arrays into proto-messages
					var tempFinalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage
					err = protojson.Unmarshal(tempFinalTestInstructionExecutionResultMessageAsByteArray, tempFinalTestInstructionExecutionResultMessage)
					if err != nil {
						common_config.Logger.WithFields(logrus.Fields{
							"Id":    "e3081dc2-c5f8-4167-8234-dec87d86b461",
							"Error": err,
							"tempMessageReceivedByWrongExecutionInstance.MessageAsJsonString": tempMessageReceivedByWrongExecutionInstance.MessageAsJsonString,
						}).Error("Something went wrong when converting 'tempProcessTestInstructionExecutionResponseStatusMessageAsByteArray' into proto-message")

						// Drop this message and continue with next message
						continue
					}

					// When there exist a 'finalTestInstructionExecutionResultMessage' then "Simulate" that
					// a gRPC-call was made from Worker, regarding 'ReportCompleteTestInstructionExecutionResult'
					if tempFinalTestInstructionExecutionResultMessage != nil {
						common_config.Logger.WithFields(logrus.Fields{
							"Id":                                   "bdd0d791-dbc1-4ebe-b423-0a8cb3c41c73",
							"common_config.ApplicationRuntimeUuid": common_config.ApplicationRuntimeUuid,
							"tempFinalTestInstructionExecutionResultMessage": tempFinalTestInstructionExecutionResultMessage,
						}).Debug("Found 'FinalTestInstructionExecutionResultMessage' in database that belongs to this 'ApplicationRuntimeUuid'")

						// Create Message to be sent to TestInstructionExecutionEngine
						var channelCommandMessage testInstructionExecutionEngine.ChannelCommandStruct
						channelCommandMessage = testInstructionExecutionEngine.ChannelCommandStruct{
							ChannelCommand: testInstructionExecutionEngine.ChannelCommandProcessFinalTestInstructionExecutionResultMessage,
							FinalTestInstructionExecutionResultMessage: tempFinalTestInstructionExecutionResultMessage,
						}

						// Send Message to TestInstructionExecutionEngine via channel
						*testInstructionExecutionEngine.TestInstructionExecutionEngineObject.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage
					}

				case common_config.ProcessTestInstructionExecutionResponseStatus:
					// The response that a Connector har received a TestInstructionExecution, to be execution

					// Convert json-strings into byte-arrays
					var tempProcessTestInstructionExecutionResponseStatusMessageAsByteArray []byte
					tempProcessTestInstructionExecutionResponseStatusMessageAsByteArray = []byte(tempMessageReceivedByWrongExecutionInstance.MessageAsJsonString)

					// Convert json-byte-arrays into proto-messages
					var tempProcessTestInstructionExecutionResponseStatusMessage *fenixExecutionServerGrpcApi.ProcessTestInstructionExecutionResponseStatus
					err = protojson.Unmarshal(tempProcessTestInstructionExecutionResponseStatusMessageAsByteArray, tempProcessTestInstructionExecutionResponseStatusMessage)
					if err != nil {
						common_config.Logger.WithFields(logrus.Fields{
							"Id":    "78482133-a267-45d6-a43a-2c3f2ad6d540",
							"Error": err,
							"tempMessageReceivedByWrongExecutionInstance.MessageAsJsonString": tempMessageReceivedByWrongExecutionInstance.MessageAsJsonString,
						}).Error("Something went wrong when converting 'tempProcessTestInstructionExecutionResponseStatusMessageAsByteArray' into proto-message")

						// Drop this message and continue with next message
						continue
					}

					// When there exist a 'tempProcessTestInstructionExecutionResponseStatusMessage' then "Simulate"
					// that a gRPC-call was made from Worker, regarding 'ProcessResponseTestInstructionExecution'
					if tempProcessTestInstructionExecutionResponseStatusMessage != nil {
						common_config.Logger.WithFields(logrus.Fields{
							"Id":                                   "788dea5f-31e1-4145-ad76-fd5e420ad230",
							"common_config.ApplicationRuntimeUuid": common_config.ApplicationRuntimeUuid,
							"tempProcessTestInstructionExecutionResponseStatusMessage": tempProcessTestInstructionExecutionResponseStatusMessage,
						}).Debug("Found 'ProcessTestInstructionExecutionResponseStatusMessage' in database that belongs to this 'ApplicationRuntimeUuid'")

						// Create Message to be sent to TestInstructionExecutionEngine
						var channelCommandMessage testInstructionExecutionEngine.ChannelCommandStruct
						channelCommandMessage = testInstructionExecutionEngine.ChannelCommandStruct{
							ChannelCommand: testInstructionExecutionEngine.ChannelCommandProcessTestInstructionExecutionResponseStatus,
							ProcessTestInstructionExecutionResponseStatus: tempProcessTestInstructionExecutionResponseStatusMessage,
						}

						// Send Message to TestInstructionExecutionEngine via channel
						*testInstructionExecutionEngine.TestInstructionExecutionEngineObject.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage
					}

				default:
					// Unknown MessageTyp, can happen when new message type appears but is missing in this 'switch'
					common_config.Logger.WithFields(logrus.Fields{
						"Id": "42411d3a-e957-431c-ad34-89eb6079ef9c",
						"tempMessageReceivedByWrongExecutionInstance": tempMessageReceivedByWrongExecutionInstance,
					}).Error("Unknown MessageType from message stored in Database")

					// Drop this message and continue with next message
					continue

				}
			}

		}

	}
}
