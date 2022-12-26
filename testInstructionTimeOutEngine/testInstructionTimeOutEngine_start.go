package testInstructionTimeOutEngine

import (
	"FenixExecutionServer/common_config"
	"github.com/sirupsen/logrus"
)

// InitiateTestInstructionExecutionTimeOutEngineChannelReader
// Initiate the channel reader which is used for sending commands to TimeOutEngine for TestInstructionExecutions
func (testInstructionExecutionTimeOutEngineObject *TestInstructionTimeOutEngineObjectStruct) InitiateTestInstructionExecutionTimeOutEngineChannelReader() {

	common_config.Logger.WithFields(logrus.Fields{
		"id":                               "4076cea7-3252-496d-9b3d-83488e699a3b",
		"NumberOfParallellTimeOutChannels": common_config.NumberOfParallellTimeOutChannels,
	}).Info("Will use parallell TimeOutChannels")

	// Initate 'nextUpcomingObjectMapKeyWithTimeOutSlice'
	nextUpcomingObjectMapKeyWithTimeOutSlice = make([]string, common_config.NumberOfParallellTimeOutChannels)

	// Create one instance per execution track
	for executionTrackNumber := 0; executionTrackNumber < common_config.NumberOfParallellTimeOutChannels; executionTrackNumber++ {

		// Initiate maps with ongoing Timers
		var timeOutMapSliceElement map[string]*timeOutMapStruct
		timeOutMapSliceElement = make(map[string]*timeOutMapStruct)

		// Add to Slice
		timeOutMapSlice = append(timeOutMapSlice, &timeOutMapSliceElement)

		// Initiate maps with Timers that TimedOut
		var timedOutMapSliceElement map[string]*timeOutMapStruct
		timedOutMapSliceElement = make(map[string]*timeOutMapStruct)

		// Add to Slice
		timedOutMapSlice = append(timedOutMapSlice, &timedOutMapSliceElement)

		// Initiate the Allocation-map for TimeOut-timers
		var allocatedTimeOutTimerMapSliceElement map[string]string
		allocatedTimeOutTimerMapSliceElement = make(map[string]string)

		// Add to Slice
		AllocatedTimeOutTimerMapSlice = append(AllocatedTimeOutTimerMapSlice, allocatedTimeOutTimerMapSliceElement)

		// Create one 'TimeOutChannelEngineCommandChannel' per execution track
		var timeOutChannelEngineCommandChannel common_config.TimeOutChannelEngineType
		timeOutChannelEngineCommandChannel = make(common_config.TimeOutChannelEngineType, timeOutChannelSize)

		TimeOutChannelEngineCommandChannelSlice = append(TimeOutChannelEngineCommandChannelSlice,
			timeOutChannelEngineCommandChannel)

		common_config.TimeOutChannelEngineCommandChannelReferenceSlice = append(
			common_config.TimeOutChannelEngineCommandChannelReferenceSlice,
			&timeOutChannelEngineCommandChannel)

		// Create one 'CancellableTimer' per execution track
		var cancellableTimer *common_config.CancellableTimerStruct
		cancellableTimer = common_config.NewCancellableTimer()
		cancellableTimerSlice = append(cancellableTimerSlice, cancellableTimer)
	}

	// Start Channel reader(s)
	for executionTrackNumber := 0; executionTrackNumber < common_config.NumberOfParallellTimeOutChannels; executionTrackNumber++ {

		go testInstructionExecutionTimeOutEngineObject.startTimeOutChannelReader(executionTrackNumber)
	}

}
