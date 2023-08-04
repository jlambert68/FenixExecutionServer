package broadcastEngine_TestInstructionNotHandledByThisInstance

import (
	"FenixExecutionServer/common_config"
	"context"
	"encoding/json"
	fenixSyncShared "github.com/jlambert68/FenixSyncShared"
	"github.com/sirupsen/logrus"
	"log"
)

// BroadcastEngineChannelSize
// The size of the channel
const BroadcastEngineChannelSize = 500

// BroadcastEngineChannelWarningLevel
// The size of warning level for the channel
const BroadcastEngineChannelWarningLevel = 400

var BroadcastEngineMessageChannel BroadcastEngineMessageChannelType

type BroadcastEngineMessageChannelType chan BroadcastingMessageForTestInstructionExecutionsStruct

type BroadcastingMessageForTestInstructionExecutionsStruct struct {
	OriginalMessageCreationTimeStamp string                                           `json:"originalmessagecreationtimestamp"`
	TestInstructionExecutions        []TestInstructionExecutionBroadcastMessageStruct `json:"testinstructionsexecutions"`
}

type TestInstructionExecutionBroadcastMessageStruct struct {
	TestInstructionExecutionUuid    string `json:"testinstructionexecutionuuid"`
	TestInstructionExecutionVersion string `json:"testinstructionexecutionversion"`
}

var err error

func InitiateAndStartBroadcastNotifyEngine_TestInstructionNotHandledByThisInstance() {

	BroadcastEngineMessageChannel = make(chan BroadcastingMessageForTestInstructionExecutionsStruct, BroadcastEngineChannelSize)
	var broadcastingMessageForExecutions BroadcastingMessageForTestInstructionExecutionsStruct
	var broadcastingMessageForExecutionsAsByteSlice []byte
	var broadcastingMessageForExecutionsAsByteSliceAsString string
	var err error
	var channelSize int

	for {

		broadcastingMessageForExecutions = <-BroadcastEngineMessageChannel

		// If size of Channel > 'BroadcastEngineChannelWarningLevel' then log Warning message
		channelSize = len(BroadcastEngineMessageChannel)
		if channelSize > BroadcastEngineChannelWarningLevel {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":                                 "92359ec0-8e17-4203-be07-ac97e2a95a42",
				"channelSize":                        channelSize,
				"BroadcastEngineChannelWarningLevel": BroadcastEngineChannelWarningLevel,
				"BroadcastEngineChannelSize":         BroadcastEngineChannelSize,
			}).Warning("Number of messages on queue for 'BroadcastEngineMessageChannel'(broadcastingEngine_ExecutionStatusUpdate) has reached a critical level")
		}

		// Create json as string
		broadcastingMessageForExecutionsAsByteSlice, err = json.Marshal(broadcastingMessageForExecutions)
		broadcastingMessageForExecutionsAsByteSliceAsString = string(broadcastingMessageForExecutionsAsByteSlice)

		common_config.Logger.WithFields(logrus.Fields{
			"id": "5c9019a5-1b97-4d5f-97a3-79977f6aa824",
			"broadcastingMessageForExecutionsAsByteSliceAsString": broadcastingMessageForExecutionsAsByteSliceAsString,
		}).Debug("Trying to send Message over Broadcast system (broadcastingEngine_ExecutionStatusUpdate)")

		_, err = fenixSyncShared.DbPool.Exec(context.Background(), "SELECT pg_notify('testInstructionNotHandledByThisInstance', $1)", broadcastingMessageForExecutionsAsByteSlice)
		if err != nil {
			common_config.Logger.WithFields(logrus.Fields{
				"Id":    "4f399c14-663e-4ece-8f2d-f3c68aa4ab2b",
				"Error": err,
				"broadcastingMessageForExecutionsAsByteSliceAsString": broadcastingMessageForExecutionsAsByteSliceAsString,
			}).Error("Number of messages on queue for 'BroadcastEngineMessageChannel'(broadcastingEngine_ExecutionStatusUpdate) has reached a critical level")

			log.Println("error sending notification:", err)
		}

	}
}
