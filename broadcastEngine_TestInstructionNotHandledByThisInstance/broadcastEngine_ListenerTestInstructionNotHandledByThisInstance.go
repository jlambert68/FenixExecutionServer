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
	"google.golang.org/protobuf/types/known/timestamppb"
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
		}).Debug("Got Broadcast message from Postgres Databas")

		err = json.Unmarshal([]byte(notification.Payload), &broadcastingMessageForExecutions)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":  "b363640f-502f-4ef5-bdd2-f57b54f824e8",
				"err": err,
			}).Error("Got some error when Unmarshal incoming json over Broadcast system")
		} else {

			// Break down 'broadcastingMessageForExecutions' and send correct content to correct sSubscribers.
			convertToChannelMessageAndPutOnChannels(broadcastingMessageForExecutions)

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
			SendID: "4d545fda-d9e4-4d35-b8af-4bbbbacf971e",
		}

		// Send message on TimeOutEngineChannel to get information about if TestInstructionExecution is handled by this Execution-instance
		*common_config.TimeOutChannelEngineCommandChannelReferenceSlice[executionTrackNumber] <- tempTimeOutChannelCommand

		// Response from TimeOutEngine
		var timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue common_config.TimeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceStruct

		// Wait for response from TimeOutEngine
		timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue =
			<-timeOutResponseChannelForIsThisHandledByThisExecutionInstance

		// Verify if TestInstructionExecution is handled by this Execution-instance
		if timeOutResponseChannelForVerifyIfTestInstructionIsHandledByThisInstanceValue.
			TestInstructionIsHandledByThisExecutionInstance == true {
			// *** TestInstructionExecution is handled by this Execution-instance ***

			// Load TestInstructionExecution from database and then remove data in database
			var finalTestInstructionExecutionResultMessage *fenixExecutionServerGrpcApi.FinalTestInstructionExecutionResultMessage
			var messagesReceivedByWrongExecutionInstance []*common_config.TestInstructionExecutionMessageReceivedByWrongExecutionStruct
			messagesReceivedByWrongExecutionInstance, err = prepareLoadTestInstructionExecutionMessagesReceivedByWrongInstance(
				tempTestInstructionExecution.TestInstructionExecutionUuid,
				int32(tempTestInstructionVersion))

			if err != nil {
				common_config.Logger.WithFields(logrus.Fields{
					"Id":  "26fa6d54-d825-4b1b-89a3-012fc2bbd1c9",
					"err": err,
					"tempTestInstructionExecution.TestInstructionExecutionUuid":    tempTestInstructionExecution.TestInstructionExecutionUuid,
					"tempTestInstructionExecution.TestInstructionExecutionVersion": tempTestInstructionExecution.TestInstructionExecutionVersion,
				}).Error("Problems when reading TestInstructionExecution from database. Dropping TestInstructionExecution")

				continue
			}

			// Convert 'tempTestInstructionExecutionEndTimeStamp' into gRPC-version
			tempFinalTestInstructionExecutionResultMessage.TestInstructionExecutionEndTimeStamp =
				timestamppb.New(tempTestInstructionExecutionEndTimeStamp)

			// Convert 'tempTestInstructionExecutionStatus' into gRPC-version
			tempFinalTestInstructionExecutionResultMessage.TestInstructionExecutionStatus =
				fenixExecutionServerGrpcApi.TestInstructionExecutionStatusEnum(tempTestInstructionExecutionStatus)

			// If there are more than one message(shouldn't be so) then add it to slice of messages
			tempFinalTestInstructionExecutionResultMessages = append(tempFinalTestInstructionExecutionResultMessages,
				&tempFinalTestInstructionExecutionResultMessage)

			// When there exist a 'finalTestInstructionExecutionResultMessage' then "Simulate" that a gRPC-call was made from Worker, regarding 'ReportCompleteTestInstructionExecutionResult'
			if finalTestInstructionExecutionResultMessage != nil {
				common_config.Logger.WithFields(logrus.Fields{
					"Id":                                   "ff0d86e2-63d2-4118-ac9b-5b18a5d70cde",
					"common_config.ApplicationRuntimeUuid": common_config.ApplicationRuntimeUuid,
					"finalTestInstructionExecutionResultMessage": finalTestInstructionExecutionResultMessage,
				}).Debug("Found TeTestInstructionExecution in database that belongs to this 'ApplicationRuntimeUuid'")

				// Create Message to be sent to TestInstructionExecutionEngine
				channelCommandMessage := testInstructionExecutionEngine.ChannelCommandStruct{
					ChannelCommand: testInstructionExecutionEngine.ChannelCommandProcessFinalTestInstructionExecutionResultMessage,
					FinalTestInstructionExecutionResultMessage: finalTestInstructionExecutionResultMessage,
				}

				// Send Message to TestInstructionExecutionEngine via channel
				*testInstructionExecutionEngine.TestInstructionExecutionEngineObject.CommandChannelReferenceSlice[executionTrackNumber] <- channelCommandMessage

			}

		}
	}
}
