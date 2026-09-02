package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// maxBodyBytes bounds what a comment, description, wiki page or batch input
// read from standard input can be, so a stream that never ends cannot exhaust
// memory.
const maxBodyBytes = 4 << 20

// bodyWording is what a command calls the text it is reading, in the errors it
// returns for it. It exists because `issue comment` has always named it a
// comment while the others say body, and the code is part of what a script
// branches on, so it cannot change underneath one of them.
type bodyWording struct {
	emptyCode    string
	emptyMessage string
	readContext  string
}

var (
	genericBody = bodyWording{emptyCode: "empty_body", emptyMessage: "body cannot be empty", readContext: "read body"}
	commentBody = bodyWording{emptyCode: "empty_comment", emptyMessage: "comment body cannot be empty", readContext: "read comment body"}
)

// readBody returns the text a command was given, whether that came from a flag,
// a file or standard input. Whitespace alone counts as nothing, which can only
// be known after reading a file.
func readBody(input io.Reader, body, bodyFile string, wording bodyWording) (string, error) {
	if bodyFile == "" {
		if strings.TrimSpace(body) == "" {
			return "", validationError(wording.emptyCode, wording.emptyMessage)
		}
		return body, nil
	}
	var data []byte
	var err error
	if bodyFile == "-" {
		data, err = io.ReadAll(io.LimitReader(input, maxBodyBytes))
	} else {
		data, err = os.ReadFile(bodyFile)
	}
	if err != nil {
		return "", fmt.Errorf("%s: %w", wording.readContext, err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return "", validationError(wording.emptyCode, wording.emptyMessage)
	}
	return string(data), nil
}
