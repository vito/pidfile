package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type Runner struct {
	Filename string
}

type ProcessExistsError struct {
	Filename string
	Pid      int
}

func (err ProcessExistsError) Error() string {
	return fmt.Sprintf("pidfile '%s' contains active pid: %d", err.Filename, err.Pid)
}

func (runner *Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	err := os.MkdirAll(filepath.Dir(runner.Filename), 0755)
	if err != nil {
		return err
	}

	pidfile, err := os.OpenFile(runner.Filename, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	err = syscall.Flock(int(pidfile.Fd()), syscall.LOCK_NB|syscall.LOCK_EX)
	if err != nil {
		return err
	}

	var existingPid int
	_, err = fmt.Fscanf(pidfile, "%d", &existingPid)
	if err == nil {
		process, err := os.FindProcess(existingPid)
		if err == nil {
			err := process.Signal(syscall.Signal(0))
			if err == nil {
				process.Release()

				return ProcessExistsError{
					Filename: runner.Filename,
					Pid:      existingPid,
				}
			}
		}
	}

	err = pidfile.Truncate(0)
	if err != nil {
		return err
	}

	_, err = pidfile.WriteAt([]byte(fmt.Sprintf("%d", os.Getpid())), 0)
	if err != nil {
		return err
	}

	close(ready)

	<-signals

	return os.Remove(runner.Filename)
}
