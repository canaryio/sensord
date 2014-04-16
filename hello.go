package main

import (
	"fmt"
	"os/exec"
)

func main() {
	curlyCmd := exec.Command("curly", "https://github.com")

	cmdOut, err := curlyCmd.Output()
	if err != nil {
		panic(err)
	}
	fmt.Print(string(cmdOut))
}
