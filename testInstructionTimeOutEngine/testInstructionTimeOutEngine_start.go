package testInstructionTimeOutEngine

import "FenixExecutionServer/common_config"

// InitiateTestInstructionExecutionTimeOutEngineChannelReader
// Initiate the channel reader which is used for sending commands to TimeOutEngine for TestInstructionExecutions
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) InitiateTestInstructionExecutionTimeOutEngineChannelReader() {

	// Initiate map with ongoing Timers
	timeOutMap = make(map[string]*timeOutMapStruct)

	// Initiate map with Timers that TimedOut
	timedOutMap = make(map[string]*timeOutMapStruct)

	// Initiate the Allocation-map for TimeOut-timers
	allocatedTimeOutTimerMap = make(map[string]string)

	// Initiate engine channel and save to reference for all to use
	TimeOutChannelEngineCommandChannel = make(chan common_config.TimeOutChannelCommandStruct, timeOutChannelSize)
	common_config.TimeOutChannelEngineCommandChannelReference = &TimeOutChannelEngineCommandChannel

	cancellableTimer = common_config.NewCancellableTimer()

	// Start Channel reader
	go testInstructionExecutionTimeOutEngineObject.startTimeOutChannelReader()

}
