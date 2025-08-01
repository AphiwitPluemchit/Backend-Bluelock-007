package test

import (
	"fmt"
	"testing"
	"time"
)

// TestTimer is a utility for measuring test execution time
type TestTimer struct {
	start time.Time
	name  string
}

// NewTestTimer creates a new test timer
func NewTestTimer(name string) *TestTimer {
	return &TestTimer{
		start: time.Now(),
		name:  name,
	}
}

// Stop stops the timer and prints the duration
func (t *TestTimer) Stop() time.Duration {
	duration := time.Since(t.start)
	fmt.Printf("⏱️  %s took %v\n", t.name, duration)
	return duration
}

// BenchmarkTest runs a test function and measures its performance
func BenchmarkTest(t *testing.T, testName string, testFunc func()) {
	timer := NewTestTimer(testName)
	defer timer.Stop()

	testFunc()
}

// BenchmarkTestWithSetup runs a test with setup and teardown functions
func BenchmarkTestWithSetup(t *testing.T, testName string, setup func(), testFunc func(), teardown func()) {
	if setup != nil {
		setup()
	}

	if teardown != nil {
		defer teardown()
	}

	BenchmarkTest(t, testName, testFunc)
}

// PerformanceAssertion checks if a test meets performance requirements
func PerformanceAssertion(t *testing.T, testName string, duration time.Duration, maxDuration time.Duration) {
	if duration > maxDuration {
		t.Errorf("❌ %s performance test failed: took %v, expected less than %v", testName, duration, maxDuration)
	} else {
		t.Logf("✅ %s performance test passed: took %v (under %v limit)", testName, duration, maxDuration)
	}
}

// TestResult represents the result of a test with timing information
type TestResult struct {
	Name     string
	Duration time.Duration
	Passed   bool
	Error    error
}

// TestSuiteResult represents the results of multiple tests
type TestSuiteResult struct {
	SuiteName   string
	TotalTests  int
	PassedTests int
	FailedTests int
	TotalTime   time.Duration
	AverageTime time.Duration
	Results     []TestResult
}

// NewTestSuiteResult creates a new test suite result
func NewTestSuiteResult(suiteName string) *TestSuiteResult {
	return &TestSuiteResult{
		SuiteName: suiteName,
		Results:   make([]TestResult, 0),
	}
}

// AddResult adds a test result to the suite
func (tsr *TestSuiteResult) AddResult(result TestResult) {
	tsr.Results = append(tsr.Results, result)
	tsr.TotalTests++
	tsr.TotalTime += result.Duration

	if result.Passed {
		tsr.PassedTests++
	} else {
		tsr.FailedTests++
	}

	tsr.AverageTime = tsr.TotalTime / time.Duration(tsr.TotalTests)
}

// PrintSummary prints a summary of the test suite results
func (tsr *TestSuiteResult) PrintSummary() {
	fmt.Printf("\n📊 Test Suite Summary: %s\n", tsr.SuiteName)
	fmt.Printf("   Total Tests: %d\n", tsr.TotalTests)
	fmt.Printf("   Passed: %d ✅\n", tsr.PassedTests)
	fmt.Printf("   Failed: %d ❌\n", tsr.FailedTests)
	fmt.Printf("   Total Time: %v\n", tsr.TotalTime)
	fmt.Printf("   Average Time: %v\n", tsr.AverageTime)
	fmt.Printf("   Success Rate: %.2f%%\n", float64(tsr.PassedTests)/float64(tsr.TotalTests)*100)

	if len(tsr.Results) > 0 {
		fmt.Printf("\n📋 Individual Test Results:\n")
		for _, result := range tsr.Results {
			status := "✅"
			if !result.Passed {
				status = "❌"
			}
			fmt.Printf("   %s %s: %v", status, result.Name, result.Duration)
			if result.Error != nil {
				fmt.Printf(" (Error: %v)", result.Error)
			}
			fmt.Println()
		}
	}
	fmt.Println()
}
