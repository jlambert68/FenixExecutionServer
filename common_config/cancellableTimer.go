package common_config

import (
	"github.com/sirupsen/logrus"
	"time"
)

type CancellableTimerEndStatusType uint8

const (
	CancellableTimerEndStatusTimedOut CancellableTimerEndStatusType = iota
	CancellableTimerEndStatusWasCancelled
)

// CancellableTimerReturnChannelType
// Channel used for response from CancellableTimer when the timer has ended or has been cancelled
type CancellableTimerReturnChannelType chan CancellableTimerEndStatusType

type CancellableTimerStruct struct {
	cancel chan bool
}

func NewCancellableTimer() *CancellableTimerStruct {
	return &CancellableTimerStruct{
		cancel: make(chan bool),
	}
}

// internal wait goroutine wrapping time.After
func (c *CancellableTimerStruct) wait(d time.Duration, ch chan bool) {
	select {
	case <-time.After(d):
		ch <- true
	case <-c.cancel:
		ch <- false
	}
}

// After mimics time.After but returns bool to signify whether we timed out or cancelled
func (c *CancellableTimerStruct) After(d time.Duration) chan bool {
	ch := make(chan bool)
	go c.wait(d, ch)
	return ch
}

// Cancel makes all the waiters receive false
func (c *CancellableTimerStruct) Cancel() {
	close(c.cancel)

}

// StartCancellableTimer
// Start a CancellableTimer
func StartCancellableTimer(t *CancellableTimerStruct,
	sleepDuration time.Duration,
	cancellableTimerReturnChannelReference *CancellableTimerReturnChannelType) {
	select {
	// timedOut will signify a timeout or cancellation
	case timedOut := <-t.After(sleepDuration):
		if timedOut {
			// When Timer times out
			Logger.WithFields(logrus.Fields{
				"id":            "8bea6fc7-9b7b-490f-8794-212f5aa24c74",
				"sleepDuration": sleepDuration,
			}).Debug("Timer did time out")

			// Send Response over channel to initiator
			*cancellableTimerReturnChannelReference <- CancellableTimerEndStatusTimedOut

		} else {
			// When Timer is cancelled
			Logger.WithFields(logrus.Fields{
				"id":            "e513f786-9632-4553-9177-624e5012ffb8",
				"sleepDuration": sleepDuration,
			}).Debug("Timer was cancelled")

			// Send Response over channel to initiator
			*cancellableTimerReturnChannelReference <- CancellableTimerEndStatusWasCancelled

		}
	}
}

/* How to use
t1 := NewCancellableTimer()
	t2 := NewCancellableTimer()
	t3 := NewCancellableTimer()
	go MyTimer(t1, time.Second*2)
	go MyTimer(t2, time.Second*4)
	go MyTimer(t3, time.Second*6)
	time.Sleep(5 * time.Second)
	t1.Cancel()
	t2.Cancel()
	t3.Cancel()

	time.Sleep(8 * time.Second)
	....
Time out!
Time out!
Cancelled!

Program exited.
*/
