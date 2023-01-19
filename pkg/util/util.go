package util

import (
	"bytes"
	"log"
	"os"
	"os/exec"
)

func Fzf(options []string) (result []string) {
	nullbyte := []byte{0}

	// execute fzf, pass it options, return the selected options
	cmd := exec.Command(
		"fzf",
		"--read0",
		"--print0",
		"--preview", "./jira \"$(echo {} | cut -d' ' -f1)\"",
		"--preview-window=down,60%",
		"--height=50%",
	)
	cmd.Stderr = os.Stderr // fzf displays its interface over stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	// asynchronously write stdin options to fzf
	go func() {
		defer stdin.Close()
		for _, o := range options {
			// todo: ignore io errors for now, crashing is fine
			stdin.Write([]byte(o))
			stdin.Write(nullbyte)
		}
	}()

	// get fzf result
	out, err := cmd.Output()
	if err != nil {
		if err.(*exec.ExitError).ExitCode() == 130 {
			return result // 130 is intentional exit from fzf, return nothing
		}
		log.Fatal(err)
	}

	// trim off the trailing null from fzf before splitting
	out = bytes.TrimRight(out, "\x00")
	for _, line := range bytes.Split(out, nullbyte) {
		result = append(result, string(line))
	}
	return result
}
