// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// package gitoperations
// The package wraps the git command line executable and parses the output to provide an interface.
package gitoperations

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
)

type LoggingInfo struct {
	suppressStdOut bool
	suppressStdErr bool
	trace          bool
	tracePrefix    string
	traceFn        func(format string, v ...interface{})
}

type Controller interface {
	RunSuppliedExecutableWithArgs(commandandargs []string) error
	WhichGit() (string, error)
	GetBranch() (string, error)
	GetRefForHead() (string, error)
	// GetUpstreamForRef is useful for extracting the tracking branch.
	GetUpstreamForRef(ref string) (string, error)
	// Deprecated: Use GetUpstreamForRef
	GetTrackingBranch() (string, error)
	HasUncommittedChanges() bool
	RefIsAheadBehind(ref string) (ahead int, behind int, err error)
	// Deprecated: use instead: RefIsAheadBehind
	BranchIsAheadOfOrigin(branch string) (bool, string, error)
	IsInsideAGitWorkingTree() (bool, error)
	GetTopLevel() (string, error)
	GetParentCommit() (string, error)
	GetHeadCommit() (string, error)
	CountCommitsWithGtOneParent(currentBranch string, ancestorCommit string) (int, error)
	GetMergeBase(parentCommit string, targetBranch string) (string, error)
	GetGraphToHead(currentBranch string, mergeTarget string, numLines int) (string, error)
	GetLastCommitOnBranch(branch string) (string, error)
	GetGlobalConfigSetting(setting string) (string, error)
	GetConfigSetting(setting string) (string, error)
	GitCanExecute() error
}

type realController struct{}

func MakeController() Controller {
	return new(realController)
}

func (Controller *realController) RunSuppliedExecutableWithArgs(commandandargs []string) error {
	return RunSuppliedExecutableWithArgs(exec.Command, commandandargs)
}

func (Controller *realController) WhichGit() (string, error) {
	return exec.LookPath("git")
}

func (Controller *realController) GetTopLevel() (string, error) {
	return GetTopLevel(exec.Command)
}

func (Controller *realController) IsInsideAGitWorkingTree() (bool, error) {
	return IsInsideAGitWorkingTree(exec.Command)
}

func (Controller *realController) GetBranch() (string, error) {
	return GetBranch(exec.Command)
}

func (Controller *realController) GetRefForHead() (string, error) {
	return GetRefForHead(exec.Command)
}

func (Controller *realController) GetHeadCommit() (string, error) {
	return GetHeadCommit(exec.Command)
}

func (Controller *realController) GetMergeBase(parentCommit string, targetBranch string) (string, error) {
	return GetMergeBase(exec.Command, parentCommit, targetBranch)
}

func (Controller *realController) GetParentCommit() (string, error) {
	return GetParentCommit(exec.Command)
}

// Deprecated: Use GetUpstreamForRef instead.
func (Controller *realController) GetTrackingBranch() (string, error) {
	return GetTrackingBranch(exec.Command)
}

func (Controller *realController) HasUncommittedChanges() bool {
	return HasUncommittedChanges(exec.Command)
}

func (Controller *realController) RefIsAheadBehind(ref string) (int, int, error) {
	return RefIsAheadBehind(exec.Command, ref)
}

func (Controller *realController) BranchIsAheadOfOrigin(branch string) (bool, string, error) {
	return BranchIsAheadOfOrigin(exec.Command, branch)
}

func (Controller *realController) GetUpstreamForRef(ref string) (string, error) {
	return GetUpstreamForRef(exec.Command, ref)
}

func (Controller *realController) GetGlobalConfigSetting(setting string) (string, error) {
	return GetGlobalConfigSetting(exec.Command, setting)
}

func (Controller *realController) GetConfigSetting(setting string) (string, error) {
	return GetConfigSetting(exec.Command, setting)
}

func (Controller *realController) GitCanExecute() error {
	return GitCanExecute(exec.Command)
}

func (Controller *realController) GetLastCommitOnBranch(branch string) (string, error) {
	return GetLastCommitOnBranch(exec.Command, branch)
}

func (Controller *realController) CountCommitsWithGtOneParent(currentBranch string, ancestorCommit string) (int, error) {
	return CountCommitsWithGtOneParent(exec.Command, currentBranch, ancestorCommit)
}

func (Controller *realController) GetGraphToHead(currentBranch string, mergeTarget string, numLines int) (string, error) {
	return GetGraphToHead(exec.Command, currentBranch, mergeTarget, numLines)
}

var (
	executableName = path.Base(os.Args[0])
	loggingInfo    = LoggingInfo{tracePrefix: "Running: ", traceFn: log.Printf}
)

func SetTrace(trace bool) {
	loggingInfo.trace = trace
}

func GetTrace() bool {
	return loggingInfo.trace
}

func maybeTrace(cmds []string) {
	if loggingInfo.trace {
		loggingInfo.traceFn(loggingInfo.tracePrefix+"%s\n", strings.Join(cmds, " "))
	}
}

// Many of these operations pass in os.exec.Command to facilitate mocking.
type Executor func(string, ...string) *exec.Cmd

func RunLoudly(cmd *exec.Cmd) error {
	// Runs the passed command, with stdout and stderr passed through to subprocess.
	// returns the error condition.
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func RunSuppliedExecutableWithArgs(exec Executor, command []string) error {
	maybeTrace(command)
	cmd := exec(command[0], command[1:]...)
	return RunLoudly(cmd)
}

func runAndGetCombinedOutput(exec Executor, cmdArr []string) (output []byte, err error) {
	maybeTrace(cmdArr)
	output, err = exec(cmdArr[0], cmdArr[1:]...).CombinedOutput()
	return
}

func scanAndSplit(output []byte) *bufio.Scanner {
	scanner := bufio.NewScanner(bytes.NewReader(output))
	scanner.Split(bufio.ScanLines)
	return scanner
}

func GetBranch(exec Executor) (string, error) {
	cmdArr := []string{"git", "rev-parse", "--abbrev-ref", "HEAD"}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return string(out), err
	}

	scanner := scanAndSplit(out)

	if !scanner.Scan() {
		return "", errors.New("Could not find branch in output string.")
	}
	line := scanner.Text()
	return strings.TrimSpace(line), nil
}

func GetRefForHead(exec Executor) (string, error) {
	// Example: when working in mainline branch, returns "refs/head/mainline"
	cmdArr := []string{"git", "symbolic-ref", "-q", "HEAD"}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return strings.TrimSpace(string(out)),
			fmt.Errorf("Could not identify upstream for ref %s: %v", "HEAD", err)
	}
	scanner := scanAndSplit(out)

	if !scanner.Scan() {
		return "", errors.New("Could not identify branch in output string.")
	}
	line := scanner.Text()
	return strings.TrimSpace(line), nil
}

func GetUpstreamForRef(exec Executor, ref string) (string, error) {
	cmdArr := []string{"git", "for-each-ref", "--format=%(upstream:short)", ref}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("Unable to identify upstream for %s: %v", ref, err)
	}
	scanner := scanAndSplit(out)
	if !scanner.Scan() {
		return "", errors.New("Could not identify upstream for ref " + ref)
	}
	line := strings.TrimSpace(scanner.Text())
	if len(line) == 0 {
		return line, errors.New("Unable to determine upstream for ref " + ref)
	}
	return line, nil
}

// Deprecated: Use GetUpstreamForRef instead.
func GetTrackingBranch(exec Executor) (string, error) {
	// Parses the output of command 'git branch -vv] to extract the tracking branch of the current branch
	// returns
	// string: the branch name
	// error: error if tracking branch is not found, or any other error state
	// when tracking branch is not found returns "" as tracking branch name

	cmdArr := []string{"git", "branch", "-vv"}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return "", err
	}
	ReForBranch, err := regexp.Compile(`^\*\s`)
	if err != nil {
		return "", err
	}
	ReForUpstream, err := regexp.Compile(`^\*\s+\S+\s+\S+\s+\[([^\]:]+).*`)
	if err != nil {
		return "", err
	}
	scanner := scanAndSplit(out)
	for scanner.Scan() {
		line := scanner.Text()
		matched := []string{}
		if matched = ReForUpstream.FindStringSubmatch(line); matched == nil {
			if matched = ReForBranch.FindStringSubmatch(line); matched != nil {
				return "", errors.New("Current branch has no upstream: " + line)
			}
			continue
		}
		return matched[1], nil
	}
	return "", errors.New("Unable to locate upstream branch")
}

func HasUncommittedChanges(exec Executor) bool {
	// Pre: User has already ascertained that the CWD is within a git workspace
	// Checks whether the git workspace has uncomitted changes, returning boolean

	// Diffing against HEAD is what guarantees a check against last commit.
	// otherwise it the diff will be against what is staged, and this could
	// lead to forgetting to commit changes before merging.
	cmdArr := []string{"git", "diff", "HEAD", "--exit-code"}
	if _, err := runAndGetCombinedOutput(exec, cmdArr); err != nil {
		return true
	}
	return false
}

func RefIsAheadBehind(exec Executor, ref string) (ahead int, behind int, err error) {
	// Example ref argument refs/heads/mainline
	// Pre: ref has a tracking branch
	// Returns the number of commits the branch is ahead and behind the tracking branch
	// On error, returns 0 for ahead and behind.

	// example strings to parse:
	//[ahead 1, behind 1]
	cmdArr := []string{"git", "for-each-ref", "--format=\"%(upstream:track)\"", ref}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return
	}
	reForAhead, _ := regexp.Compile(`.*\[.*ahead (\d+).*\].*`)
	reForBehind, _ := regexp.Compile(`.*\[.*behind (\d+).*\].*`)
	if err != nil {
		return
	}
	line := ""
	hasLine := false
	scanner := scanAndSplit(out)
	if scanner.Scan() {
		line = scanner.Text()
		hasLine = true
	}
	if !hasLine {
		err = errors.New("No output while determining branch ahead/behind tracking branch.")
		return
	}
	if matched := reForAhead.FindStringSubmatch(line); matched != nil {
		ahead, _ = strconv.Atoi(matched[1])
	}
	if matched := reForBehind.FindStringSubmatch(line); matched != nil {
		behind, _ = strconv.Atoi(matched[1])
	}
	return
}

// Deprecated: Use instead RefIsAheadBehind which uses a _plumbing_ interface instead of porcelain one.
func BranchIsAheadOfOrigin(exec Executor, branch string) (bool, string, error) {
	// Parses the output of command 'git branch -vv] to see if given branch is ahead of origin.
	// returns
	// bool: whether it is ahead or not
	// proof: the number of commits it is ahead (the string that was matched)
	// error: any error that occurred causing an early (or final) return
	proof := "" // proof will be filled in when the function returns false.  In this manner, we reserve the error
	// object for error reporting only.
	cmdArr := []string{"git", "branch", "-vv"}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return false, proof, err
	}
	ReForBranchLocation, err := regexp.Compile(`^\*?\s+` + branch + `\s+.*`)
	if err != nil {
		return false, "", err
	}
	ReForHasUpstream, err := regexp.Compile(`^\*?\s+` + branch + `\s+(\S+)\s+\[(.*)`)
	if err != nil {
		return false, "", err
	}

	ReForAheadUpstream, err := regexp.Compile(`^\*?\s+` + branch + `\s+(\S+)\s+\[[^\]]+: ahead\s+([^\]]+)\]\s+.*`)
	if err != nil {
		return false, "", err
	}
	scanner := scanAndSplit(out)
	for scanner.Scan() {
		line := scanner.Text()
		if matched := ReForBranchLocation.FindStringSubmatch(line); matched == nil {
			continue
		}
		if matched := ReForHasUpstream.FindStringSubmatch(line); matched == nil {
			return false, "", errors.New("No tracking branch available.")
		}
		if matched := ReForAheadUpstream.FindStringSubmatch(line); matched != nil {
			return true, matched[2], nil
		}
		return false, "", nil
	}
	return false, "", errors.New("Unable to locate branch in git output.\nGit output:\n" + string(out))
}

// Deprecated: Checkout functionality should be access via RunSppliedExecutableWithArgs
func Checkout(exec Executor, currentBranch string, targetBranch string) error {
	cmdArr := []string{"git", "checkout", targetBranch}
	maybeTrace(cmdArr)
	if err := RunLoudly(exec(cmdArr[0], cmdArr[1:]...)); err != nil {
		return errors.New("Failed to checkout " + targetBranch + ". Repository will be left in " +
			currentBranch + " branch.")
	}
	return nil
}

// Deprecated: Fetch functionality should be accessed via RunSuppliedExecutableWithArgs
func Fetch(exec Executor, branch string) error {
	cmdArr := []string{"git", "fetch", "-p", "origin", fmt.Sprintf("%s:%s", branch, branch)}
	maybeTrace(cmdArr)
	cmd := exec(cmdArr[0], cmdArr[1:]...)
	return RunLoudly(cmd)
}

// Deprecated: Pull functionality should be accessed via RunSuppliedExecutableWithArgs
func Pull(exec Executor, srcBranch string, rebase bool) error {
	rebaseStr := ""
	if rebase {
		rebaseStr = "--rebase"
	}
	cmdArr := []string{"git", "pull", rebaseStr, ".", srcBranch}
	maybeTrace(cmdArr)
	cmd := exec(cmdArr[0], cmdArr[1:]...)
	return RunLoudly(cmd)
}

// Deprecated: ResetTarget functionality should be accessed via RunSuppliedExecutableWithArgs
func ResetTarget(exec Executor, targetBranch string) error {
	cmdArr := []string{"git", "reset", "--hard",
		fmt.Sprintf("origin/%s", targetBranch)}
	maybeTrace(cmdArr)
	cmd := exec(cmdArr[0], cmdArr[1:]...)
	return RunLoudly(cmd)
}

// Deprecated: DeleteBranch functionality should be accessed via RunSuppliedExecutableWithArgs
func DeleteBranch(exec Executor, sourceBranch string) error {
	cmdArr := []string{"git", "branch", "-D", sourceBranch}
	maybeTrace(cmdArr)
	cmd := exec(cmdArr[0], cmdArr[1:]...)
	return RunLoudly(cmd)
}

// Deprecated: MergeSourceToTarget functionality should be accessed via RunSuppliedExecutableWithArgs
func MergeSourceToTarget(exec Executor, sourceBranch string) error {
	cmdArr := []string{"git", "merge", "--squash", sourceBranch}
	maybeTrace(cmdArr)
	cmd := exec(cmdArr[0], cmdArr[1:]...)
	return RunLoudly(cmd)
}

// Deprecated: Commit functionality should be accessed via RunSuppliedExecutableWithArgs
func Commit(exec Executor) error {
	cmdArr := []string{"git", "commit"}
	cmd := exec(cmdArr[0], cmdArr[1:]...)
	return RunLoudly(cmd)
}

// Deprecated: Push functionality should be accessed via RunSuppliedExecutableWithArgs
func Push(exec Executor) error {
	cmdArr := append([]string{"git", "push"})
	maybeTrace(cmdArr)
	cmd := exec(cmdArr[0], cmdArr[1:]...)
	return RunLoudly(cmd)
}

func IsInsideAGitWorkingTree(exec Executor) (bool, error) {
	// on success returns the relative path to .git directory
	cmdArr := []string{"git", "rev-parse", "--is-inside-work-tree"}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return false, errors.New(string(out))
	}

	scanner := scanAndSplit(out)

	if !scanner.Scan() {
		return false, errors.New("No output from git command.")
	}
	line := scanner.Text()
	if line == "true" {
		return true, nil
	} else if line == "false" {
		return false, nil
	} else if strings.HasPrefix(line, "fatal: Not a git repository") {
		return false, nil
	}
	return false, errors.New("Unrecognized output: " + line)
}

func GetTopLevel(exec Executor) (string, error) {
	// On success returns the root of the git workspace.
	// Users should consider calling IsInsideGitWorkingTree before calling this function.

	cmdArr := []string{"git", "rev-parse", "--show-toplevel"}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return "", errors.New(string(out))
	}

	scanner := scanAndSplit(out)

	if !scanner.Scan() {
		return "", errors.New("No output from git command.")
	}
	return strings.TrimRight(scanner.Text(), "\n"), nil
}

func GetParentCommit(exec Executor) (string, error) {
	// Returns the parent (HEAD~) commit hash.
	// Error is non-nil when the command fails.
	cmd := exec("git", "rev-parse", "HEAD~")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "Failed to identify parent commit: " + string(out), err
	}

	scanner := scanAndSplit(out)

	if !scanner.Scan() {
		return "", errors.New("No revision found.")
	}
	line := scanner.Text()
	return line, nil
}

func GetHeadCommit(exec Executor) (string, error) {
	// Returns the HEAD commit hash.
	// Error is non-nil when the command fails.
	cmdArr := []string{"git", "rev-parse", "HEAD"}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return string(out), errors.New("Failed to identify HEAD commit: " + err.Error())
	}

	scanner := scanAndSplit(out)

	if !scanner.Scan() {
		return "", errors.New("No revision found.")
	}
	line := scanner.Text()
	return line, nil
}

func CountCommitsWithGtOneParent(exec Executor, currentBranch string, ancestorCommit string) (int, error) {
	// Input: ancestorCommit is some ancestor hash of the current HEAD.
	// Returns true iff last commit has > 1 parent.
	// Error is non-nil when the command fails.
	// Having greater than one parent indicates that the last commit is not maintaining linear history, and for some
	// users that is a property to keep track of.
	cmdArr := []string{"git", "rev-list", "--count", "--min-parents=2", fmt.Sprintf("--branches=%s", currentBranch), "--ancestry-path", ancestorCommit + "..HEAD"}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return 0, errors.New("Parent Count Check: " + err.Error())
	}

	scanner := scanAndSplit(out)

	if !scanner.Scan() {
		return 0, errors.New("Failed to identify path from " + ancestorCommit + " to head.")
	}
	count, err := strconv.Atoi(scanner.Text())
	return count, err
}

func GetMergeBase(exec Executor, parentCommit string, targetBranch string) (string, error) {
	// Identify the common ancestor which will be used in the event of a merge.
	// parentCommit: Should be the sole parent of HEAD.  User is responsible for ensuring HEAD has only a single
	// parent.
	// targetBranch: The branch we would possibly merge into.  Could be: origin/mainline
	// Returns: hash of the merge base, non-nil error when an error occurs.

	cmdArray := []string{"git", "merge-base", targetBranch, parentCommit}
	maybeTrace(cmdArray)
	cmd := exec(cmdArray[0], cmdArray[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	scanner := scanAndSplit(out)
	if !scanner.Scan() {
		return "", errors.New("Failed to identify the merge base")
	}
	line := scanner.Text()
	return line, nil
}

func GetGraphToHead(exec Executor, currentBranch string, mergeTarget string, numLines int) (string, error) {
	var sb strings.Builder
	// mergeTarget will be ommitted from the output by the command below.  If mergeTarget~ is used instead, we find that extranous
	// descendencts of mergebase get output.
	cmdArr := []string{"git", "log", "--decorate", "--oneline", "--graph", "--all", fmt.Sprintf("--branches=%s", currentBranch), "--ancestry-path",
		mergeTarget + "..HEAD"}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return string(out), err
	}
	scanner := scanAndSplit(out)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if !first {
			line = "\n" + line
		}
		first = false
		sb.WriteString(line)
	}
	return sb.String(), nil
}

func GetLastCommitOnBranch(exec Executor, branch string) (string, error) {
	// Returns the last commit in given branch.
	cmdArr := []string{"git", "log", branch, "-n1", "--format=format:%H"}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return string(out), err
	}
	scanner := scanAndSplit(out)
	if !scanner.Scan() {
		return "", errors.New("Failed to identify final commit on branch.")
	}
	line := scanner.Text()
	return line, nil
}

func GetGlobalConfigSetting(exec Executor, setting string) (string, error) {
	cmdArr := []string{"git", "config", "--global", "--get", setting}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return string(out), err
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	if !scanner.Scan() {
		return "", errors.New("No setting found.")
	}
	line := scanner.Text()
	return strings.TrimSpace(string(line)), nil
}

func GetConfigSetting(exec Executor, setting string) (string, error) {
	cmdArr := []string{"git", "config", "--get", setting}
	out, err := runAndGetCombinedOutput(exec, cmdArr)
	if err != nil {
		return string(out), err
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	if !scanner.Scan() {
		return "", errors.New("No setting found.")
	}
	line := scanner.Text()
	return strings.TrimSpace(string(line)), nil
}

func GitCanExecute(exec Executor) error {
	// Simple test to make sure we can get git to execute.
	// Returns non-nil error if git can not execute a simple command.
	cmdArr := []string{"git", "config", "--list"}
	_, err := runAndGetCombinedOutput(exec, cmdArr)
	return err
}
