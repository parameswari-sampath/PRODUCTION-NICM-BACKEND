package scheduler

import (
	"log"
	"mcq-exam/live"
	"time"
)

// DummyFirstEmail simulates sending first email (conference invitation)
func DummyFirstEmail() {
	log.Printf("[%s] EXECUTING: DummyFirstEmail - Sending conference invitations to all students", time.Now().Format(time.RFC3339))
	// Simulate work
	time.Sleep(500 * time.Millisecond)
	log.Printf("[%s] COMPLETED: DummyFirstEmail - Conference invitations sent successfully", time.Now().Format(time.RFC3339))
}

// DummySecondEmail simulates sending second email (test invitation)
func DummySecondEmail() {
	log.Printf("[%s] EXECUTING: DummySecondEmail - Sending test invitations to eligible students", time.Now().Format(time.RFC3339))
	// Simulate work
	time.Sleep(500 * time.Millisecond)
	log.Printf("[%s] COMPLETED: DummySecondEmail - Test invitations sent successfully", time.Now().Format(time.RFC3339))
}

// FunctionRegistry maps function names to actual functions
var FunctionRegistry = map[string]func(){
	"DummyFirstEmail":            DummyFirstEmail,
	"DummySecondEmail":           DummySecondEmail,
	"SendFirstEmailToAll":        SendFirstEmailToAll,
	"SendSecondEmailToEligible":  SendSecondEmailToEligible,
	"Phase1FirstMailVerification": live.Phase1FirstMailVerification,
	"Phase2SecondMailSending":    live.Phase2SecondMailSending,
}

// ExecuteFunction calls a registered function by name
func ExecuteFunction(functionName string) bool {
	fn, exists := FunctionRegistry[functionName]
	if !exists {
		log.Printf("ERROR: Function '%s' not found in registry", functionName)
		return false
	}

	log.Printf("Executing function: %s", functionName)
	fn()
	return true
}
