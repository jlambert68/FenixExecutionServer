package testInstructionExecutionEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
)

// InitiateTestInstructionExecutionEngineCommandChannelReader
// Initiate the channel reader which is used for sending commands to TestInstruction Execution Engine
func (executionEngine *TestInstructionExecutionEngineStruct) InitiateTestInstructionExecutionEngineCommandChannelReader(
	executionEngineCommandChannel []ExecutionEngineChannelType) {

	// Create and start one instance per execution track for channel
	for executionTrackNumber := 0; executionTrackNumber < common_config.NumberOfParallellExecutionEngineCommandChannels; executionTrackNumber++ {

		executionEngine.CommandChannelReferenceSlice = append(
			executionEngine.CommandChannelReferenceSlice,
			&executionEngineCommandChannel[executionTrackNumber])

		go executionEngine.startCommandChannelReader(executionTrackNumber)

	}

	// Trigger TestInstructionEngine to check if there are any Zombie-TestInstructionExecutions stuck in UnderExecutions
	channelCommandMessage := ChannelCommandStruct{
		ChannelCommand:                   ChannelCommandLookForZombieTestInstructionExecutionsInUnderExecution,
		ChannelCommandTestCaseExecutions: nil,
	}

	// Send Message on Channel, use first instance
	*executionEngine.CommandChannelReferenceSlice[0] <- channelCommandMessage

	// Trigger TestInstructionEngine to check if there are any Zombie-TestInstructionExecutions, UnderExecution, that have timed out
	channelCommandMessage = ChannelCommandStruct{
		ChannelCommand:                          ChannelCommandLookForZombieTestInstructionExecutionsThatHaveTimedOut,
		ChannelCommandTestCaseExecutions:        nil,
		ChannelCommandTestInstructionExecutions: nil,
	}

	// Send Message on Channel, use first instance
	*executionEngine.CommandChannelReferenceSlice[0] <- channelCommandMessage

	// Trigger TestInstructionEngine to Load TestExecutionTimes from Database, if there are any, into TimeOutEngine
	channelCommandMessage = ChannelCommandStruct{
		ChannelCommand:                          ChannelCommandReCreateTimeOutTimersAtApplicationStartUp,
		ChannelCommandTestCaseExecutions:        nil,
		ChannelCommandTestInstructionExecutions: nil,
	}

	// Send Message on Channel, use first instance
	*executionEngine.CommandChannelReferenceSlice[0] <- channelCommandMessage

	return

}

// SetLogger
// Set to use the same Logger reference as is used by central part of system
func (executionEngine *TestInstructionExecutionEngineStruct) SetLogger(logger *logrus.Logger) {

	executionEngine.logger = logger

	return

}
