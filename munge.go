package storaged

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Munge generates a credential with the specified payload and a UID_RESTRICTION for the decryptor.
func Munge(payload string) (string, error) {
	cmd := exec.Command("munge", "-s", payload)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error running munge: %w", err)
	}
	return string(output), nil
}

// MungeOutput is the parsed output from unmunge. All fields may be nil if they were not provided.
type MungeOutput struct {
	UserID            *int
	GroupID           *int
	UserIDRestriction *int
	EncodeHost        net.IP
	EncodeTime        *time.Time
	DecodeTime        *time.Time
	Payload           []byte
}

// Unmunge is a function that invokes unmunge and parses its output to create a MungeOutput. It
// returns an error if any of its expectation of the output format is violated.
func Unmunge(encryptedPayload string) (*MungeOutput, error) {
	cmd := exec.Command(
		"unmunge", "-N", "-k",
		"ENCODE_HOST,ENCODE_TIME,DECODE_TIME,UID,GID,UID_RESTRICTION",
	)
	cmd.Stdin = bytes.NewBufferString(encryptedPayload)
	output, err := cmd.Output()
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		return nil, fmt.Errorf("error validating munge credential: %w, %s", exitErr, exitErr.Stderr)
	case err != nil:
		return nil, fmt.Errorf("error running unmunge: %w", err)
	}
	splitOutput := bytes.SplitN(output, []byte("\n\n"), 2)
	if len(splitOutput) == 1 {
		return nil, fmt.Errorf("error running unmunge: missing payload")
	}
	var parsedOutput MungeOutput
	for entry := range strings.SplitSeq(string(splitOutput[0]), "\n") {
		splitEntry := strings.SplitN(entry, ":", 2)
		splitEntry[1] = strings.TrimSpace(splitEntry[1])
		switch splitEntry[0] {
		case "ENCODE_HOST":
			parsedOutput.EncodeHost = net.ParseIP(splitEntry[1])
			if parsedOutput.EncodeHost == nil {
				return nil, fmt.Errorf("invalid ENCODE_HOST value: %s", splitEntry[1])
			}
		case "ENCODE_TIME":
			encodeTimeSec, err := strconv.ParseInt(splitEntry[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid ENCODE_TIME value: %s", splitEntry[1])
			}
			encodeTime := time.Unix(encodeTimeSec, 0)
			parsedOutput.EncodeTime = &encodeTime
		case "DECODE_TIME":
			decodeTimeSec, err := strconv.ParseInt(splitEntry[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid DECODE_TIME value: %s", splitEntry[1])
			}
			decodeTime := time.Unix(decodeTimeSec, 0)
			parsedOutput.DecodeTime = &decodeTime
		case "UID":
			uid, err := strconv.Atoi(splitEntry[1])
			if err != nil {
				return nil, fmt.Errorf("invalid UID value: %s", splitEntry[1])
			}
			parsedOutput.UserID = &uid
		case "GID":
			gid, err := strconv.Atoi(splitEntry[1])
			if err != nil {
				return nil, fmt.Errorf("invalid GID value: %s", splitEntry[1])
			}
			parsedOutput.GroupID = &gid
		case "UID_RESTRICTION":
			uidRestriction, err := strconv.Atoi(splitEntry[1])
			if err != nil {
				return nil, fmt.Errorf("invalid UID_RESTRICTION value: %s", splitEntry[1])
			}
			parsedOutput.UserIDRestriction = &uidRestriction
		default:
			// We specified the list of fields so Munge shouldn't be feeding us unknown keys.
			return nil, fmt.Errorf(
				"unexpected metadata key %q from unmunge: value %q",
				splitEntry[0], splitEntry[1],
			)
		}
	}
	parsedOutput.Payload = splitOutput[1]
	return &parsedOutput, nil
}
