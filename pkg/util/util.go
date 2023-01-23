package util

import (
	"bytes"
	"fmt"
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
		"--preview", fmt.Sprintf("%s \"$(echo {} | cut -d' ' -f1)\"", os.Args[0]),
		"--preview-window=down,70%",
		"--height=80%",
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
			_, err = stdin.Write([]byte(o))
			if err != nil {
				log.Fatal(err)
			}
			_, err = stdin.Write(nullbyte)
			if err != nil {
				log.Fatal(err)
			}
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
