package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type AnsibleExecutor struct {
	PlaybookPath string
	Target       Target
}

type Target struct {
	Host string
	User string
}

type CommandTimeoutError struct {
	Timeout time.Duration
}

func (e *CommandTimeoutError) Error() string {
	return fmt.Sprintf(
		"command timed out after %s",
		e.Timeout,
	)
}

func NewAnsibleExecutor(
	playbookPath string,
	target Target,
) *AnsibleExecutor {
	return &AnsibleExecutor{
		PlaybookPath: playbookPath,
		Target:       target,
	}
}

type ansibleJSONOutput struct {
	Plays []struct {
		Tasks []struct {
			Task struct {
				Name string `json:"name"`
			} `json:"task"`
			Hosts map[string]struct {
				Stdout string `json:"stdout"`
				RC     int    `json:"rc"`
				Failed bool   `json:"failed"`
			} `json:"hosts"`
		} `json:"tasks"`
	} `json:"plays"`
}

// Run executes a single read-only command on the configured target.
func (a *AnsibleExecutor) Run(
	cmd string,
	timeout time.Duration,
) (string, error) {
	if timeout <= 0 {
		return "", fmt.Errorf("timeout must be greater than zero")
	}

	if timeout > 60*time.Second {
		timeout = 60 * time.Second
	}

	inventoryPath, cleanup, err := a.inventoryFile()
	if err != nil {
		return "", err
	}
	defer cleanup()

	extraVars := map[string]interface{}{
		"target_host":     "diagnos-target",
		"cmd":             cmd,
		"command_timeout": fmt.Sprintf("%ds", int(timeout.Seconds())),
	}

	extraVarsJSON, err := json.Marshal(extraVars)
	if err != nil {
		return "", fmt.Errorf(
			"failed to marshal ansible variables: %w",
			err,
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		timeout,
	)
	defer cancel()

	args := []string{
		"-i",
		inventoryPath,
		a.PlaybookPath,
		"-e",
		string(extraVarsJSON),
	}

	command := exec.CommandContext(
		ctx,
		"ansible-playbook",
		args...,
	)

	command.Env = append(
		os.Environ(),
		"ANSIBLE_STDOUT_CALLBACK=json",
	)

	var stdout, stderr bytes.Buffer

	command.Stdout = &stdout
	command.Stderr = &stderr

	err = command.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return "", &CommandTimeoutError{
			Timeout: timeout,
		}
	}

	if err != nil {
		return "", fmt.Errorf(
			"ansible-playbook failed: %w (stderr: %s)",
			err,
			stderr.String(),
		)
	}

	var parsed ansibleJSONOutput

	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		return "", fmt.Errorf(
			"failed to parse ansible json output: %w",
			err,
		)
	}

	if len(parsed.Plays) == 0 || len(parsed.Plays[0].Tasks) == 0 {
		return "", fmt.Errorf(
			"no task results returned by ansible",
		)
	}

	lastTask := parsed.Plays[0].Tasks[len(parsed.Plays[0].Tasks)-1]

	result, ok := lastTask.Hosts["diagnos-target"]
	if !ok {
		return "", fmt.Errorf(
			"no result found for target %s",
			a.Target.Host,
		)
	}

	if result.Failed || result.RC != 0 {
		return "", fmt.Errorf(
			"command failed on %s (rc=%d): %s",
			a.Target.Host,
			result.RC,
			result.Stdout,
		)
	}

	return result.Stdout, nil
}

func (a *AnsibleExecutor) CheckTarget() error {
	_, err := a.Run(
		"true",
		10*time.Second,
	)

	if err != nil {
		return fmt.Errorf(
			"target %q is not reachable through ansible: %w",
			a.Target.Host,
			err,
		)
	}

	return nil
}

func (a *AnsibleExecutor) inventoryFile() (string, func(), error) {
	inventory := fmt.Sprintf(
		"[target]\n%s ansible_host=%s ansible_user=%s\n",
		"diagnos-target",
		a.Target.Host,
		a.Target.User,
	)

	file, err := os.CreateTemp(
		"",
		"diagnos-inventory-*.ini",
	)
	if err != nil {
		return "", nil, fmt.Errorf(
			"create ansible inventory: %w",
			err,
		)
	}

	cleanup := func() {
		_ = os.Remove(file.Name())
	}

	if _, err := file.WriteString(inventory); err != nil {
		_ = file.Close()
		cleanup()

		return "", nil, fmt.Errorf(
			"write ansible inventory: %w",
			err,
		)
	}

	if err := file.Close(); err != nil {
		cleanup()

		return "", nil, fmt.Errorf(
			"close ansible inventory: %w",
			err,
		)

	}

	return file.Name(), cleanup, nil
}
