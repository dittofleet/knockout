package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/dittofleet/knockout/internal/app"
	"github.com/dittofleet/knockout/internal/power"
)

// Daemon listens for messages through crosstalk until crosstalk stops
// listening, which launchd answers by starting it again.
func Daemon() error {
	listen := exec.Command("crosstalk", "listen", app.Name)
	listen.Stderr = os.Stderr
	replies, err := listen.StdinPipe()
	if err != nil {
		return err
	}
	messages, err := listen.StdoutPipe()
	if err != nil {
		return err
	}
	if err := listen.Start(); err != nil {
		return err
	}
	log.Printf("listening for %q", app.Name)
	serve(messages, replies, power.Ready, power.Shutdown)
	// Usually crosstalk has exited already. If reading stopped first, it
	// would otherwise go on listening with nobody answering.
	listen.Process.Kill()
	return listen.Wait()
}

// delivery is a line from `crosstalk listen`: a message, or a notice from
// crosstalk itself.
type delivery struct {
	ID         string          `json:"id"`
	From       string          `json:"from"`
	Body       json.RawMessage `json:"body"`
	WantsReply bool            `json:"wantsReply"`
	Crosstalk  json.RawMessage `json:"crosstalk"`
}

type reply struct {
	ID      string `json:"id"`
	Body    string `json:"body,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// grace is how long the reply has to reach the sender before the shutdown
// takes crosstalk down with everything else.
var grace = 2 * time.Second

var errUnknown = errors.New(`expected {"do":"shutdown"} or {"do":"check"}`)

// serve answers each message read from messages, writing replies, until
// messages ends.
func serve(messages io.Reader, replies io.Writer, ready, shutdown func() error) {
	lines := bufio.NewScanner(messages)
	lines.Buffer(nil, 1<<20)
	for lines.Scan() {
		var msg delivery
		if err := json.Unmarshal(lines.Bytes(), &msg); err != nil {
			log.Printf("unreadable line from crosstalk: %v", err)
			continue
		}
		if msg.Crosstalk != nil {
			log.Printf("crosstalk: %s", msg.Crosstalk)
			continue
		}

		var body struct {
			Do string `json:"do"`
		}
		// A body that does not parse is refused, even when part of it did.
		if json.Unmarshal(msg.Body, &body) != nil {
			body.Do = ""
		}
		log.Printf("%q from %s", body.Do, msg.From)
		answer := func(text string, err error) {
			if err != nil {
				log.Printf("refused: %v", err)
			}
			if !msg.WantsReply {
				return
			}
			r := reply{ID: msg.ID, Body: text}
			if err != nil {
				r.Error, r.Message = "refused", err.Error()
			}
			line, _ := json.Marshal(r)
			replies.Write(append(line, '\n'))
		}

		switch body.Do {
		case "check":
			answer("ready", ready())
		case "shutdown":
			if err := ready(); err != nil {
				answer("", err)
				continue
			}
			answer("shutting down", nil)
			time.Sleep(grace)
			log.Printf("shutting down, as %s asked", msg.From)
			if err := shutdown(); err != nil {
				log.Printf("could not shut down: %v", err)
			}
		default:
			answer("", errUnknown)
		}
	}
}
