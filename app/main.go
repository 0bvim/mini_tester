package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// TestCase represents a single shell command test case
type TestCase struct {
	Command        string `json:"command"`
	Description    string `json:"description"`
	ExpectedOutput string `json:"expected_output,omitempty"`
	ExpectedError  string `json:"expected_error,omitempty"`
	ExpectedCode   int    `json:"expected_code,omitempty"`
}

// TestCases represents the JSON structure for test cases
type TestCases struct {
	Tests []TestCase `json:"test_cases"`
}

// TestResult stores the results of a single test
type TestResult struct {
	Description         string `json:"description"`
	BashOutput          string `json:"bash_output"`
	MinishellOutput     string `json:"minishell_output"`
	BashError           string `json:"bash_error"`
	MinishellError      string `json:"minishell_error"`
	BashReturnCode      int    `json:"bash_return_code"`
	MinishellReturnCode int    `json:"minishell_return_code"`
	OutputMatch         bool   `json:"output_match"`
	ErrorMatch          bool   `json:"error_match"`
	ReturnCodeMatch     bool   `json:"return_code_match"`
	ExpectedOutputMatch bool   `json:"expected_output_match"`
	ExpectedErrorMatch  bool   `json:"expected_error_match"`
	ExpectedCodeMatch   bool   `json:"expected_code_match"`
}

// ShellTester handles shell command testing
type ShellTester struct {
	bashPath      string
	minishellPath string
}

// NewShellTester creates a new ShellTester instance
func NewShellTester(bashPath, minishellPath string) (*ShellTester, error) {
	if _, err := os.Stat(bashPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("bash executable not found at %s", bashPath)
	}
	if _, err := os.Stat(minishellPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("minishell executable not found at %s", minishellPath)
	}
	return &ShellTester{bashPath: bashPath, minishellPath: minishellPath}, nil
}

// runCommand executes a command in the specified shell
func (st *ShellTester) runCommand(shellPath, command string) (string, string, int) {
	cmd := exec.Command(shellPath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err.Error(), 1
	}

	if err := cmd.Start(); err != nil {
		return "", err.Error(), 1
	}

	_, err = stdin.Write([]byte(command + "\nexit\n"))
	if err != nil {
		return "", err.Error(), 1
	}
	_ = stdin.Close()

	err = cmd.Wait()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), exitCode
}

// compareOutput compares output between bash and minishell
func (st *ShellTester) compareOutput(testCases []TestCase) map[string]TestResult {
	results := make(map[string]TestResult)

	for _, tc := range testCases {
		bashOut, bashErr, bashRC := st.runCommand(st.bashPath, tc.Command)
		miniOut, miniErr, miniRC := st.runCommand(st.minishellPath, tc.Command)

		results[tc.Command] = TestResult{
			Description:         tc.Description,
			BashOutput:          bashOut,
			MinishellOutput:     miniOut,
			BashError:           bashErr,
			MinishellError:      miniErr,
			BashReturnCode:      bashRC,
			MinishellReturnCode: miniRC,
			OutputMatch:         bashOut == miniOut,
			ErrorMatch:          bashErr == miniErr,
			ReturnCodeMatch:     bashRC == miniRC,
			ExpectedOutputMatch: tc.ExpectedOutput == "" || miniOut == tc.ExpectedOutput,
			ExpectedErrorMatch:  tc.ExpectedError == "" || miniErr == tc.ExpectedError,
			ExpectedCodeMatch:   tc.ExpectedCode == 0 || miniRC == tc.ExpectedCode,
		}
	}

	return results
}

// generateDiff generates detailed differences for mismatched outputs
func (st *ShellTester) generateDiff(results map[string]TestResult) map[string]string {
	differences := make(map[string]string)
	dmp := diffmatchpatch.New()

	for cmd, result := range results {
		if !result.OutputMatch || !result.ErrorMatch || !result.ReturnCodeMatch {
			diffs := dmp.DiffMain(result.BashOutput, result.MinishellOutput, false)
			differences[cmd] = dmp.DiffPrettyText(diffs)
		}
	}

	return differences
}

// loadTestCases loads test cases from a JSON file
func loadTestCases(filepath string) ([]TestCase, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	var testCases TestCases
	if err := json.Unmarshal(data, &testCases); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %v", err)
	}

	return testCases.Tests, nil
}

func main() {
	bashPath := flag.String("bash", "/bin/bash", "Path to Bash executable")
	minishellPath := flag.String("minishell", "./minishell", "Path to Minishell executable")
	testsPath := flag.String("tests", "test_cases.json", "Path to test cases JSON file")
	outputPath := flag.String("output", "", "Path to save test results JSON file")
	flag.Parse()

	// Load test cases
	testCases, err := loadTestCases(*testsPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error loading test cases: %v\n", err)
		os.Exit(1)
	}

	// Initialize tester
	tester, err := NewShellTester(*bashPath, *minishellPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	results := tester.compareOutput(testCases)
	differences := tester.generateDiff(results)

	// Calculate statistics
	totalTests := len(results)
	passedTests := 0
	for _, r := range results {
		if r.OutputMatch && r.ErrorMatch && r.ReturnCodeMatch {
			passedTests++
		}
	}

	// Print summary
	fmt.Printf("\nTest Summary (%d/%d passed):\n", passedTests, totalTests)
	fmt.Println(strings.Repeat("=", 50))

	for cmd, result := range results {
		status := "PASS"
		if !result.OutputMatch || !result.ErrorMatch || !result.ReturnCodeMatch {
			status = "FAIL"
		}
		fmt.Printf("\nTest: %s\n", result.Description)
		fmt.Printf("Command: %s\n", cmd)
		fmt.Printf("Status: %s\n", status)
	}

	// Print detailed differences
	if len(differences) > 0 {
		fmt.Printf("\nDetailed Differences:\n")
		fmt.Println(strings.Repeat("=", 50))
		for cmd, diff := range differences {
			fmt.Printf("\nTest: %s\n", results[cmd].Description)
			fmt.Printf("Command: %s\n", cmd)
			fmt.Printf("\nDifferences detected:\n%s\n", diff)
		}
	}

	// Save results if output path provided
	if *outputPath != "" {
		outputData := struct {
			Summary struct {
				TotalTests  int `json:"total_tests"`
				PassedTests int `json:"passed_tests"`
				FailedTests int `json:"failed_tests"`
			} `json:"summary"`
			Results     map[string]TestResult `json:"results"`
			Differences map[string]string     `json:"differences"`
		}{
			Summary: struct {
				TotalTests  int `json:"total_tests"`
				PassedTests int `json:"passed_tests"`
				FailedTests int `json:"failed_tests"`
			}{
				TotalTests:  totalTests,
				PassedTests: passedTests,
				FailedTests: totalTests - passedTests,
			},
			Results:     results,
			Differences: differences,
		}

		jsonData, err := json.MarshalIndent(outputData, "", "  ")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error creating JSON output: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(*outputPath, jsonData, 0644); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nDetailed results saved to %s\n", *outputPath)
	}
}
