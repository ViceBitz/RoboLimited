package tools

/*
Writes to a file
*/

import (
	"log"
	"os"
)

func WriteLineToFile(fileName string, line string) {
	// Open or create the file for writing (append if it exists)
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, _ = f.WriteString(line + "\n")
}
