// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package gitoperations

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

var traceCounter int

func fakePrintf(formatstring string, rest ...interface{}) {
	traceCounter += 1
}

func setup() {
	SetTrace(false)
	traceCounter = 0
	loggingInfo = LoggingInfo{traceFn: fakePrintf}
}

// Provides a utility function to help mock execution of a command line executable.
// A parent process encodes the desired stdout and exit status behavior in environment variables STDOUT and EXIT_STATUS
// so the TestExecCommandHelper sub-process knows how to behave.
func TestExecCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, os.Getenv("STDOUT"))
	i, _ := strconv.Atoi(os.Getenv("EXIT_STATUS"))
	os.Exit(i)
}

// Wraps TestExecCommandHelper with custom output to mock execution of a command line executable.
// The arguments stdErrorOut and exitStatus are passed to the TestExecCommandHelper executable via environment
// variables so the mock knows how to behave for the test.
func createFakeExecCommand(stdErrorOut string, exitStatus int) Executor {
	return func(command string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestExecCommandHelper", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		es := strconv.Itoa(exitStatus)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1",
			"STDOUT=" + stdErrorOut,
			"EXIT_STATUS=" + es}
		return cmd
	}
}

func TestCheckout(t *testing.T) {
	{
		setup()
		mockExec := createFakeExecCommand("foo", 0)
		err := Checkout(mockExec, "current_branch", "target_branch")
		if err != nil {
			t.Fatalf("Expected nil error, but received %v", err)
		}
	}
	{
		mockExec := createFakeExecCommand("foo", 1)
		err := Checkout(mockExec, "current_branch", "target_branch")
		if err == nil {
			t.Fatalf("Expected non-nil error")
		} else if !strings.HasPrefix(err.Error(), "Failed to checkout ") {
			t.Fatalf("Unexpected error: %v", err)
		}
	}
}

func TestGetParentCommitSuccess(t *testing.T) {
	setup()
	expectedParent := "f4035569c97a051f56798adecf2facb744bbf969"
	mockExec := createFakeExecCommand(expectedParent+"\n\n", 0)
	commit, err := GetParentCommit(mockExec)
	if err != nil {
		t.Errorf("Expected nil error, but got: %v", err)
	}
	if commit != expectedParent {
		t.Errorf("Expected '%s' but received '%s'", expectedParent, commit)
	}
}

func TestGetParentCommitBareRepo(t *testing.T) {
	// Simulates execution of the GetParentCommit on brand new repo with no parent commit.
	setup()
	fakeMessage := `HEAD~
fatal: ambiguous argument 'HEAD~': unknown revision or path not in the working tree.
Use '--' to separate paths from revisions, like this:
'git <command> [<revision>...] -- [<file>...]'`

	mockExec := createFakeExecCommand(fakeMessage, 128)
	message, err := GetParentCommit(mockExec)
	if err == nil {
		t.Fatalf("Expected non-nil error.")
	}
	if !strings.HasPrefix(message, "Failed to identify parent commit:") {
		t.Fatalf("Unexpected message %s: ", message)
	}
}

func TestGetParentCommitParseError(t *testing.T) {
	setup()
	// Simulates execution of the GetParentCommit on brand new repo with no parent commit.
	fakeMessage := ""
	mockExec := createFakeExecCommand(fakeMessage, 0)
	_, err := GetParentCommit(mockExec)
	if err == nil {
		t.Fatalf("Expected non-nil error.")
	}
	if !strings.HasPrefix(err.Error(), "No revision found.") {
		t.Fatalf("Unexpected message %v: ", err)
	}
}

func TestGetHeadCommitSuccess(t *testing.T) {
	setup()
	expectedParent := "f4035569c97a051f56798adecf2facb744bbf969"
	mockExec := createFakeExecCommand(expectedParent+"\n\n", 0)
	commit, err := GetHeadCommit(mockExec)
	if err != nil {
		t.Errorf("Expected nil error, but got: %v", err)
	}
	if commit != expectedParent {
		t.Errorf("Expected '%s' but received '%s'", expectedParent, commit)
	}
}

func TestGetHeadCommitNonZeroExit(t *testing.T) {
	setup()
	expectedError := "Failed to identify HEAD commit: "
	mockExec := createFakeExecCommand("f4035569c97a051f56798adecf2facb744bbf969\n", 1)
	_, err := GetHeadCommit(mockExec)
	if err == nil {
		t.Errorf("Expected non-nil error.")
	}
	if !strings.HasPrefix(err.Error(), expectedError) {
		t.Errorf("Expected '%s' but received '%v'", expectedError, err)
	}
}

func TestGetHeadCommitParseError(t *testing.T) {
	setup()
	// Simulates execution of the GetParentCommit on brand new repo with no parent commit.
	fakeMessage := ""
	mockExec := createFakeExecCommand(fakeMessage, 0)
	_, err := GetHeadCommit(mockExec)
	if err == nil {
		t.Fatalf("Expected non-nil error.")
	}
	if !strings.HasPrefix(err.Error(), "No revision found.") {
		t.Fatalf("Unexpected message %v: ", err)
	}
}

func TestCountCommitsWithGtOneParent0(t *testing.T) {
	setup()
	expectedParent := "0"
	mockExec := createFakeExecCommand(expectedParent+"\n", 0)
	count, err := CountCommitsWithGtOneParent(mockExec, "current", "f4035569c97a051f56798adecf2facb744bbf969")
	if err != nil {
		t.Errorf("Expected nil error, but got: %v", err)
	}
	if strconv.Itoa(count) != "0" {
		t.Errorf("Expected '%s' but received '%d'", expectedParent, count)
	}
}

func TestCountCommitsWithGtOneParentGitError(t *testing.T) {
	setup()
	expectedParent := "0"
	mockExec := createFakeExecCommand(expectedParent+"\n", 1)
	_, err := CountCommitsWithGtOneParent(mockExec, "current", "f4035569c97a051f56798adecf2facb744bbf969")
	if err == nil {
		t.Errorf("Expected non-nil error.")
	}
	expectedError := "Parent Count Check: exit status 1"
	if err.Error() != expectedError {
		t.Errorf("Expected '%s' but received '%v'", expectedError, err)
	}
}

func TestGetParentCommitNoOutput(t *testing.T) {
	setup()
	// Simulates execution of the GetParentCommit on brand new repo with no parent commit.
	fakeMessage := ""
	mockExec := createFakeExecCommand(fakeMessage, 0)
	_, err := CountCommitsWithGtOneParent(mockExec, "current", "f4035569c97a051f56798adecf2facb744bbf969")
	if err == nil {
		t.Fatalf("Expected non-nil error.")
	}
	expectedOutput := "Failed to identify path from f4035569c97a051f56798adecf2facb744bbf969 to head."
	if !strings.HasPrefix(err.Error(), expectedOutput) {
		t.Fatalf("Unexpected message %v: ", err)
	}
}

func TestGetMergeBaseSuccess(t *testing.T) {
	setup()
	expectedMergeBase := "f4035569c97a051f56798adecf2facb744bbf969"
	mockExec := createFakeExecCommand(expectedMergeBase+"\n", 0)
	actualMergeBase, err := GetMergeBase(mockExec, expectedMergeBase, "mainline")
	if err != nil {
		t.Errorf("Expected nil error, but got: %v", err)
	}
	if actualMergeBase != expectedMergeBase {
		t.Errorf("Expected '%s' but received '%s'", expectedMergeBase, actualMergeBase)
	}
}

func TestGetMergeBaseGitFailure(t *testing.T) {
	setup()
	expectedMergeBase := "f4035569c97a051f56798adecf2facb744bbf969"
	mockExec := createFakeExecCommand(expectedMergeBase+"\n", 1)
	_, err := GetMergeBase(mockExec, expectedMergeBase, "mainline")
	if err == nil {
		t.Errorf("Expected non-nil error.")
	}
	expectedError := "exit status 1"
	if err.Error() != expectedError {
		t.Errorf("Expected '%s' but received '%v'", expectedError, err)
	}
}

func TestGetMergeBaseNoMessage(t *testing.T) {
	setup()
	expectedMergeBase := ""
	mockExec := createFakeExecCommand(expectedMergeBase, 0)
	_, err := GetMergeBase(mockExec, expectedMergeBase, "mainline")
	if err == nil {
		t.Errorf("Expected non-nil error.")
	}
	expectedError := "Failed to identify the merge base"
	if err.Error() != expectedError {
		t.Errorf("Expected '%s' but received '%v'", expectedError, err)
	}
}

func TestGetGraphToHeadSuccess(t *testing.T) {
	setup()
	expectedMsg := "a\nb\nc\n"
	mockExec := createFakeExecCommand(expectedMsg, 0)
	actualMsg, err := GetGraphToHead(mockExec, "current-branch", "origin/mainline", 25)
	if err != nil {
		t.Errorf("Expected nil error, but got: %v", err)
	}
	if actualMsg+"\n" != expectedMsg {
		t.Errorf("Expected '%s' but received '%s'", expectedMsg, actualMsg)
	}
}

func TestGetGraphToHeadGitFailure(t *testing.T) {
	setup()
	expectedMsg := "a\nb\nc\n"
	mockExec := createFakeExecCommand(expectedMsg, 1)
	_, err := GetGraphToHead(mockExec, "current-branch", "origin/mainline", 25)
	if err == nil {
		t.Errorf("Expected non-nil error.")
	}
	expectedError := "exit status 1"
	if err.Error() != expectedError {
		t.Errorf("Expected '%s' but received '%v'", expectedError, err)
	}
}

func TestNoArgGitCommands(t *testing.T) {
	setup()
	type cmd struct {
		f                       func(cmd Executor) (string, error)
		successMsg              string
		expectedOutputOnSuccess string

		failOutputMsg string
		failErrorMsg  string
	}
	functions := []cmd{
		cmd{GetBranch, "aschein-dev", "aschein-dev", "", "Could not find branch in output string."},
	}
	for _, fn := range functions {
		mockSuccess := createFakeExecCommand(fn.successMsg+"\n", 0)
		output, err := fn.f(mockSuccess)
		// test successful run.
		if err != nil {
			t.Errorf("Expected nil, received: %v", err)
		} else if output != fn.expectedOutputOnSuccess {
			t.Errorf("Expected %s, but received: %s", fn.successMsg, output)
		}
		// test git command failure
		mockFail := createFakeExecCommand(fn.successMsg+"\n", 1)
		stdErrorMsg := "exit status 1"
		_, err = fn.f(mockFail)
		if err == nil {
			t.Errorf("Expected nil error, but received: %v", err)
		} else if err.Error() != stdErrorMsg {
			t.Errorf("Expected error %v but received %s", err, stdErrorMsg)
		}

		// Called not found, because this error is caused by inability to find the expected text in the output of the
		// git command
		mockNotFound := createFakeExecCommand(fn.failOutputMsg, 0)
		_, err = fn.f(mockNotFound)
		if err == nil {
			t.Errorf("Expected non-nil error")
		} else if err.Error() != fn.failErrorMsg {
			t.Errorf("Expected error: %s, but received %v", fn.failErrorMsg, err)
		}
	}
}

func TestGitPull(t *testing.T) {
	setup()
	mockSuccess := createFakeExecCommand("mocked text\n", 0)
	err := Pull(mockSuccess, "test-branch", true)
	// test successful run.
	if err != nil {
		t.Errorf("Expected nil, received: " + err.Error())
	}
	// test git command failure
	mockFail := createFakeExecCommand("mocked text for failure\n", 1)
	stdErrorMsg := "exit status 1"
	err = Pull(mockFail, "test-branch", true)
	if err == nil {
		t.Errorf("Expected nil error, but received: %v", err)
	} else if err.Error() != stdErrorMsg {
		t.Errorf("Expected error %v but received %s", err, stdErrorMsg)
	}
}

func TestSingleArgGitCommandsReturningErrorOnly(t *testing.T) {
	setup()
	type cmd struct {
		f    func(cmd Executor, arg1 string) error
		arg1 string
	}
	functions := []cmd{
		cmd{Fetch, "test-branch"},
		cmd{ResetTarget, "test-branch"},
		cmd{DeleteBranch, "test-branch"},
		cmd{MergeSourceToTarget, "test-branch"},
	}
	for _, fn := range functions {
		mockSuccess := createFakeExecCommand("mocked text\n", 0)
		err := fn.f(mockSuccess, fn.arg1)
		// test successful run.
		if err != nil {
			t.Errorf("Expected nil, received: " + err.Error())
		}
		// test git command failure
		mockFail := createFakeExecCommand("mocked text for failure\n", 1)
		stdErrorMsg := "exit status 1"
		err = fn.f(mockFail, fn.arg1)
		if err == nil {
			t.Errorf("Expected nil error, but received: %v", err)
		} else if err.Error() != stdErrorMsg {
			t.Errorf("Expected error %v but received %s", err, stdErrorMsg)
		}
	}
}

func TestNoArgGitCommandsReturningErrorOnly(t *testing.T) {
	setup()
	type cmd struct {
		f func(cmd Executor) error
	}
	functions := []cmd{
		cmd{Commit},
		cmd{Push},
	}
	for _, fn := range functions {
		mockSuccess := createFakeExecCommand("mocked text\n", 0)
		err := fn.f(mockSuccess)
		// test successful run.
		if err != nil {
			t.Errorf("Expected nil, received: " + err.Error())
		}
		// test git command failure
		mockFail := createFakeExecCommand("mocked text for failure\n", 1)
		stdErrorMsg := "exit status 1"
		err = fn.f(mockFail)
		if err == nil {
			t.Errorf("Expected nil error, but received: %v", err)
		} else if err.Error() != stdErrorMsg {
			t.Errorf("Expected error %v but received %s", err, stdErrorMsg)
		}
	}
}

func TestIsInsideAGitWorkingTree(t *testing.T) {
	setup()
	{ // Some git commands emit the text "true" or "false"
		mockTrue := createFakeExecCommand("true\n", 0)
		outcome, err := IsInsideAGitWorkingTree(mockTrue)
		// test successful run.
		if err != nil {
			t.Errorf("Expected nil, received: %v", err)
		} else if !outcome {
			t.Errorf("Expected true, but received false")
		}
	}
	{
		// successful run with a false return
		mockFalse := createFakeExecCommand("false\n", 0)
		outcome, err := IsInsideAGitWorkingTree(mockFalse)
		if err != nil {
			t.Errorf("Expected nil, received: " + err.Error())
		} else if outcome {
			t.Errorf("Expected false, but received true")
		}
	}
	{
		// Is run outside a git directory => false return
		mockFalse := createFakeExecCommand("fatal: Not a git repository\n", 0)
		outcome, err := IsInsideAGitWorkingTree(mockFalse)
		if err != nil {
			t.Errorf("Expected nil, received: " + err.Error())
		} else if outcome {
			t.Errorf("Expected false, but received true")
		}
	}
	{
		// git produces an error
		mockGitFail := createFakeExecCommand("\n", 1)
		outcome, err := IsInsideAGitWorkingTree(mockGitFail)
		if err == nil {
			t.Errorf("Expected non-nil")
		} else if outcome {
			t.Errorf("Expected false, but received true")
		} else if err.Error() != "\n" {
			t.Errorf("Expected %s, but received %v", "'\n'", err)
		}
	}
	{
		// git produces un-expected output
		mockGitFail := createFakeExecCommand("foo\n", 0)
		outcome, err := IsInsideAGitWorkingTree(mockGitFail)
		expectedOutput := "Unrecognized output: foo"
		if err == nil {
			t.Errorf("Expected non-nil")
		} else if outcome {
			t.Errorf("Expected false, but received true")
		} else if err.Error() != expectedOutput {
			t.Errorf("Expected %s, but received %v", expectedOutput, err)
		}
	}
	{
		// git produces no output
		mockGitFail := createFakeExecCommand("", 0)
		outcome, err := IsInsideAGitWorkingTree(mockGitFail)
		expectedOutput := "No output from git command."
		if err == nil {
			t.Errorf("Expected non-nil")
		} else if outcome {
			t.Errorf("Expected false, but received true")
		} else if err.Error() != expectedOutput {
			t.Errorf("Expected %s, but received %v", expectedOutput, err)
		}
	}
}

func TestGetTopLevel(t *testing.T) {
	setup()
	{ // Some git commands emit the text "true" or "false"
		expected := "/my/dir"
		mockTrue := createFakeExecCommand(expected+"\n", 0)
		actual, err := GetTopLevel(mockTrue)
		// test successful run.
		if err != nil {
			t.Errorf("Expected nil, received: %v", err)
		} else if actual != "/my/dir" {
			t.Errorf("Expected '%s', but received '%s'", expected, actual)
		}
	}
	{
		// Is run outside a git directory => false return
		expected := "fatal: Not a git repository\n"
		mockFail := createFakeExecCommand(expected, 1)
		_, err := GetTopLevel(mockFail)
		if err == nil {
			t.Errorf("Expected non-nil.")
		} else if err.Error() != expected {
			t.Errorf("Expected '%s', but received '%v'", expected, err)
		}
	}
	{
		// git produces an error
		mockGitFail := createFakeExecCommand("", 0)
		actual, err := GetTopLevel(mockGitFail)
		if err == nil {
			t.Errorf("Expected non-nil")
		} else if actual != "" {
			t.Errorf("Expected null string for actual, but received '%s'", actual)
		} else if !strings.Contains(err.Error(), "No output from git command.") {
			t.Errorf("Expected %s, but received %v", "'\n'", err)
		}
	}
}

func TestHasUncommittedChanges(t *testing.T) {
	setup()
	{ // exit non-zero => true
		mockGit1 := createFakeExecCommand("", 1)
		outcome := HasUncommittedChanges(mockGit1)
		if !outcome {
			t.Errorf("Expected true.")
		}
	}
	{ // exit 0 => false
		mockGit0 := createFakeExecCommand("", 0)
		outcome := HasUncommittedChanges(mockGit0)
		if outcome {
			t.Errorf("Expected false.")
		}
	}
}

func TestRefIsAheadBehind(t *testing.T) {
	setup()
	{
		mockGit := createFakeExecCommand("[ahead 2, behind 3]\n", 0)
		ahead, behind, err := RefIsAheadBehind(mockGit, "ref")
		if err != nil {
			t.Fatalf("Expected nil error, but received: %q", err)
		}
		if ahead != 2 || behind != 3 {
			t.Fatalf("Expected 2,3 but received %d,%d", ahead, behind)
		}
	}
	{
		mockGit := createFakeExecCommand("\n", 0)
		ahead, behind, err := RefIsAheadBehind(mockGit, "ref")
		if err != nil {
			t.Fatalf("Expected nil error, but received: %q", err)
		}
		if ahead != 0 || behind != 0 {
			t.Fatalf("Expected 0,0 but received %d,%d", ahead, behind)
		}
	}
	{
		mockGit := createFakeExecCommand("[ahead 2]\n", 0)
		ahead, behind, err := RefIsAheadBehind(mockGit, "ref")
		if err != nil {
			t.Fatalf("Expected nil error, but received: %q", err)
		}
		if ahead != 2 || behind != 0 {
			t.Fatalf("Expected 2,0 but received %d,%d", ahead, behind)
		}
	}
	{
		mockGit := createFakeExecCommand("[behind 3]\n", 0)
		ahead, behind, err := RefIsAheadBehind(mockGit, "ref")
		if err != nil {
			t.Fatalf("Expected nil error, but received: %q", err)
		}
		if ahead != 0 || behind != 3 {
			t.Fatalf("Expected 0,3 but received %d,%d", ahead, behind)
		}
	}
	{
		mockGit := createFakeExecCommand("[behind 3]\n", 1)
		ahead, behind, err := RefIsAheadBehind(mockGit, "ref")
		if err == nil {
			t.Fatalf("Expected non-nil error.")
		}
		if ahead != 0 || behind != 0 {
			t.Errorf("Expected 0,0 but received %d,%d", ahead, behind)
		}
		if !strings.HasPrefix("exit status 1.", err.Error()) {
			t.Fatalf("Expected 'exit status 1.', but received '%v'", err)
		}
	}
}

func TestTargetIsAheadOfOriginGitFails(t *testing.T) {
	setup()
	// Git exits with non-zero status
	mockGitFail := createFakeExecCommand("foo", 1)
	outcome, _, err := BranchIsAheadOfOrigin(mockGitFail, "target_branch")
	if outcome {
		t.Errorf("Expected false.")
	}
	if err == nil {
		t.Fatalf("Expected non-nil error.")
	}
	expected := "exit status 1"
	if err.Error() != expected {
		t.Fatalf("Expected: '%s', but received '%v'", expected, err)
	}
}

func TestTargetIsAheadOfOriginTrackingMissing(t *testing.T) {
	setup()
	mockGit := createFakeExecCommand("* mainline my comment", 0)
	outcome, _, err := BranchIsAheadOfOrigin(mockGit, "mainline")
	if outcome {
		t.Errorf("Expected false.")
	}
	if err == nil {
		t.Fatalf("Expected non-nil error.")
	}
	expected := "No tracking branch available."
	if !strings.HasPrefix(err.Error(), expected) {
		t.Fatalf("Expected prefix: '%s', but received '%v'", expected, err)
	}
}

func TestBranchIsAheadOfOriginTrue(t *testing.T) {
	setup()
	mockGit := createFakeExecCommand("* aschein-dev  96be17e [origin/mainline] Tiering\n  mainline    68e43cb8b [origin/mainline: ahead 1] Tiering", 0)
	outcome, message, err := BranchIsAheadOfOrigin(mockGit, "mainline")
	expectedMessage := "1"
	if !outcome {
		t.Errorf("Expected false.")
	}
	if err != nil {
		t.Fatalf("Expected nil error, but received '%v'", err)
	}
	if message != expectedMessage {
		t.Fatalf("Expected message '%s', but received: '%s'", expectedMessage, message)
	}

}

func TestBranchIsAheadOfOriginFalse(t *testing.T) {
	setup()
	mockGit := createFakeExecCommand(" aschein-dev [origin/mainline] Tiering\n mainline    68e43cb8b [origin/mainline] Tiering", 0)
	outcome, _, err := BranchIsAheadOfOrigin(mockGit, "mainline")
	if outcome {
		t.Errorf("Expected false.")
	}
	if err != nil {
		t.Fatalf("Expected nil error, but received '%v'", err)
	}
}

func TestRunExecutable(t *testing.T) {
	setup()
	{
		sucessExec := createFakeExecCommand("Mocked Exec", 0)
		err := RunSuppliedExecutableWithArgs(sucessExec, []string{"a", "b", "c"})
		if err != nil {
			t.Fatalf("Expected nil error, but received '%v'", err)
		}
	}
	{
		failExec := createFakeExecCommand("Mocked Exec", 1)
		err := RunSuppliedExecutableWithArgs(failExec, []string{"a", "b", "c"})
		if err == nil {
			t.Fatalf("Expected non-nil error, but received '%v'", err)
		}
		expected := "exit status 1"
		if err.Error() != expected {
			t.Fatalf("Expected '%s', but received '%v'", expected, err)
		}
	}
}

func TestTrace(t *testing.T) {
	setup()

	if GetTrace() != false {
		t.Fatalf("expected true")
	}
	maybeTrace([]string{"first log"})
	if traceCounter != 0 {
		t.Fatalf("Expected traceCounter == 0, but instead: %d", traceCounter)
	}

	SetTrace(true)
	if GetTrace() != true {
		t.Fatalf("expected false")
	}
	if loggingInfo.trace != true {
		t.Fatalf("logignInfo.trace should be true")
	}
	maybeTrace([]string{"foo", "bar"})
	if traceCounter != 1 {
		t.Fatalf("Expected traceCounter == 1, but instead: %d", traceCounter)
	}
}

func TestGetLastCommitSucceeds(t *testing.T) {
	setup()
	// Git exits with non-zero status
	mockGetLastCommit := createFakeExecCommand("foo", 0)
	outcome, err := GetLastCommitOnBranch(mockGetLastCommit, "target_branch")
	if err != nil {
		t.Errorf("Expected nil erorr, but got %v", err)
	}
	expected := "foo"
	if outcome != expected {
		t.Fatalf("Expected: '%s', but received '%s'", expected, outcome)
	}
}

func TestGetLastCommitFails(t *testing.T) {
	setup()
	// Git exits with non-zero status
	mockGetLastCommit := createFakeExecCommand("foo", 1)
	_, err := GetLastCommitOnBranch(mockGetLastCommit, "target_branch")
	if err == nil {
		t.Errorf("Expected non-nil erorr")
	}
	expected := "exit status 1"
	if err.Error() != expected {
		t.Fatalf("Expected: '%s', but received '%v'", expected, err)
	}
}

func TestGetTrackingBranch(t *testing.T) {
	setup()
	{ // This example has a tracking branch ending with ']'
		output := `  aschein-dev2 96be17e rename files and add operations.
* mainline     01b37f4 [origin/mainline] Adding function to determine the latest commit on a branch
  aschein0dev  a2c1bb0 [origin/mainline: behind 3] commit by Octane
  help         96be17e rename files and add operations.
`
		mockSuccess := createFakeExecCommand(output, 0)
		branch, err := GetTrackingBranch(mockSuccess)
		if err != nil {
			t.Errorf("Expected nil, received: " + err.Error())
		} else if branch != "origin/mainline" {
			t.Errorf("Expected %s, but received %s.", "origin/mainline", branch)
		}
	}
	{ // This example has a tracking branch ending with ':'
		output := `  aschein-dev2 96be17e rename files and add operations.
  mainline     01b37f4 [origin/mainline] Adding function to determine the latest commit on a branch
* aschein0dev  a2c1bb0 [origin/mainline: behind 3] commit by Octane
  help         96be17e rename files and add operations.
`
		mockSuccess := createFakeExecCommand(output, 0)
		branch, err := GetTrackingBranch(mockSuccess)
		if err != nil {
			t.Errorf("Expected nil, received: %v", err)
		} else if branch != "origin/mainline" {
			t.Errorf("Expected %s, but received %s.", "origin/mainline", branch)
		}
	}
	{ // No tracking branch
		output := `  aschein-dev2 96be17e rename files and add operations.
  mainline     01b37f4 [origin/mainline] Adding function to determine the latest commit on a branch
  aschein0dev  a2c1bb0 [origin/mainline: behind 3] commit by Octane
*  help         96be17e rename files and add operations.
`
		mockSuccess := createFakeExecCommand(output, 0)
		branch, err := GetTrackingBranch(mockSuccess)
		expected := "Current branch has no upstream:"
		if err == nil {
			t.Errorf("Expected non-nil error.")
		} else if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Expected string '%s' does not match '%v'", expected, err)
		}
		if branch != "" {
			t.Errorf("Expected nil string, but received %s.", branch)
		}
	}
	{ // No current branch
		output := `  aschein-dev2 96be17e rename files and add operations.
  mainline     01b37f4 [origin/mainline] Adding function to determine the latest commit on a branch
  aschein0dev  a2c1bb0 [origin/mainline: behind 3] commit by Octane
  help         96be17e rename files and add operations.
`
		mockSuccess := createFakeExecCommand(output, 0)
		branch, err := GetTrackingBranch(mockSuccess)
		expected := "Unable to locate upstream branch"
		if err == nil {
			t.Errorf("Expected non-nil error.")
		} else if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Expected ")
		}
		if branch != "" {
			t.Errorf("Expected nil string, but received %s.", branch)
		}
	}
}

func TestGetUpstreamForRef(t *testing.T) {
	setup()
	{ // This example has a tracking branch ending with ']'
		mockSuccess := createFakeExecCommand("origin/mainline", 0)
		branch, err := GetUpstreamForRef(mockSuccess, "foo")
		if err != nil {
			t.Errorf("Expected nil, received: %v", err)
		} else if branch != "origin/mainline" {
			t.Errorf("Expected %s, but received %s.", "origin/mainline", branch)
		}
	}
	{ // No tracking branch
		mockFailure := createFakeExecCommand("\n", 1)
		branch, err := GetUpstreamForRef(mockFailure, "foo")
		expected := "Unable to identify upstream for foo:"
		if err == nil {
			t.Errorf("Expected non-nil error.")
		} else if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Expected string '%s' does not match '%v'", expected, err)
		}
		if branch != "" {
			t.Errorf("Expected nil string, but received %s.", branch)
		}
	}
	{ // Empty string with zero exit status.
		mockFailure := createFakeExecCommand("", 0)
		branch, err := GetUpstreamForRef(mockFailure, "foo")
		expected := "Could not identify upstream for ref foo"
		if err == nil {
			t.Errorf("Expected non-nil error.")
		} else if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Expected string '%s' does not match '%v'", expected, err)
		}
		if branch != "" {
			t.Errorf("Expected nil string, but received %s.", branch)
		}
	}
}

func TestGetRefForHead(t *testing.T) {
	setup()
	{ // This example has a tracking branch ending with ']'
		expected := "refs/heads/recovered-plumbing"
		mockSuccess := createFakeExecCommand(expected, 0)
		branch, err := GetRefForHead(mockSuccess)
		if err != nil {
			t.Errorf("Expected nil, received: " + err.Error())
		} else if branch != expected {
			t.Errorf("Expected %s, but received %s.", "origin/mainline", branch)
		}
	}
	{ // No tracking branch
		mockFailure := createFakeExecCommand("\n", 1)
		branch, err := GetRefForHead(mockFailure)
		expected := "Could not identify upstream for ref "
		if err == nil {
			t.Errorf("Expected non-nil error.")
		} else if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Expected string '%s' does not match '%v'", expected, err)
		}
		if branch != "" {
			t.Errorf("Expected nil string, but received %s.", branch)
		}
	}
	{ // Empty string with zero exit status.
		mockFailure := createFakeExecCommand("", 0)
		branch, err := GetRefForHead(mockFailure)
		expected := "Could not identify branch in output string."
		if err == nil {
			t.Errorf("Expected non-nil error.")
		} else if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Expected string '%s' does not match '%v'", expected, err)
		}
		if branch != "" {
			t.Errorf("Expected nil string, but received %s.", branch)
		}
	}
}

func TestGetGlobalConfigSetting(t *testing.T) {
	setup()
	{ // Success case
		expected := "foo"
		mockSuccess := createFakeExecCommand("foo\n", 0)
		message, err := GetGlobalConfigSetting(mockSuccess, "pull.rebase")
		if err != nil {
			t.Errorf("Expected nil error, but received: %v", err)
		}
		if message != expected {
			t.Errorf("Expected '%s', but received '%s'", expected, message)
		}
	}
	{ // Failure
		expected := "bar\n"
		mockFailure := createFakeExecCommand(expected, 1)
		message, err := GetGlobalConfigSetting(mockFailure, "pull.rebase")
		if err == nil {
			t.Errorf("Expected non-nil error")
		}
		if message != expected {
			t.Errorf("Expected '%s' but received '%s'", expected, message)
		}
	}
	{ // No setting found
		expected := ""
		expectedError := "No setting found."
		mockFailure := createFakeExecCommand(expected, 0)
		message, err := GetGlobalConfigSetting(mockFailure, "pull.rebase")
		if err == nil {
			t.Errorf("Expected non-nil error")
		}
		if err.Error() != expectedError {
			t.Errorf("Expected '%s' but received '%v'", expectedError, err)
		}
		if message != expected {
			t.Errorf("Expected '%s' but received '%s'", expected, message)
		}
	}
}

func TestGetConfigSetting(t *testing.T) {
	setup()
	{ // Success case
		expected := "foo"
		mockSuccess := createFakeExecCommand("foo\n", 0)
		message, err := GetConfigSetting(mockSuccess, "pull.rebase")
		if err != nil {
			t.Errorf("Expected nil error, but received: %v", err)
		}
		if message != expected {
			t.Errorf("Expected '%s', but received '%s'", expected, message)
		}
	}
	{ // Failure
		expected := "bar\n"
		mockFailure := createFakeExecCommand(expected, 1)
		message, err := GetConfigSetting(mockFailure, "pull.rebase")
		if err == nil {
			t.Errorf("Expected non-nil error")
		}
		if message != expected {
			t.Errorf("Expected '%s' but received '%s'", expected, message)
		}
	}
	{ // No setting found
		expected := ""
		expectedError := "No setting found."
		mockFailure := createFakeExecCommand(expected, 0)
		message, err := GetConfigSetting(mockFailure, "pull.rebase")
		if err == nil {
			t.Errorf("Expected non-nil error")
		}
		if err.Error() != expectedError {
			t.Errorf("Expected '%s' but received '%v'", expectedError, err)
		}
		if message != expected {
			t.Errorf("Expected '%s' but received '%s'", expected, message)
		}
	}
}

func TestGitCanExecute(t *testing.T) {
	setup()
	{ // Success case
		mockSuccess := createFakeExecCommand("", 0)
		err := GitCanExecute(mockSuccess)
		if err != nil {
			t.Errorf("Expected nil error, but received: %v", err)
		}
	}
	{ // Failure
		mockFailure := createFakeExecCommand("", 1)
		err := GitCanExecute(mockFailure)
		if err == nil {
			t.Errorf("Expected non-nil error")
		}
	}
}
