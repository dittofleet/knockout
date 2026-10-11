package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strings"
	"testing"
)

func init() {
	grace = 0
	log.SetOutput(io.Discard)
}

type fake struct {
	ready     error
	shutdowns int
}

func (f *fake) serve(t *testing.T, lines ...string) []reply {
	t.Helper()
	var out bytes.Buffer
	serve(strings.NewReader(strings.Join(lines, "\n")), &out,
		func() error { return f.ready },
		func() error { f.shutdowns++; return nil })
	var replies []reply
	dec := json.NewDecoder(&out)
	for dec.More() {
		var r reply
		if err := dec.Decode(&r); err != nil {
			t.Fatal(err)
		}
		replies = append(replies, r)
	}
	return replies
}

func TestShutdownRepliesThenShutsDown(t *testing.T) {
	f := &fake{}
	got := f.serve(t, `{"id":"1","from":"kiwi","body":{"do":"shutdown"},"wantsReply":true}`)
	if f.shutdowns != 1 {
		t.Errorf("shut down %d times, want 1", f.shutdowns)
	}
	if len(got) != 1 || got[0] != (reply{ID: "1", Body: "shutting down"}) {
		t.Errorf("got %+v", got)
	}
}

func TestShutdownToEveryMacNeedsNoReply(t *testing.T) {
	f := &fake{}
	got := f.serve(t, `{"id":"1","from":"kiwi","body":{"do":"shutdown"}}`)
	if f.shutdowns != 1 || len(got) != 0 {
		t.Errorf("shut down %d times with replies %+v", f.shutdowns, got)
	}
}

func TestShutdownNotAllowedIsRefused(t *testing.T) {
	f := &fake{ready: errors.New("needs a password")}
	got := f.serve(t, `{"id":"1","from":"kiwi","body":{"do":"shutdown"},"wantsReply":true}`)
	if f.shutdowns != 0 {
		t.Error("shut down anyway")
	}
	if len(got) != 1 || got[0].Error != "refused" || got[0].Message != "needs a password" {
		t.Errorf("got %+v", got)
	}
}

func TestCheckOnlyChecks(t *testing.T) {
	f := &fake{}
	got := f.serve(t, `{"id":"1","from":"kiwi","body":{"do":"check"},"wantsReply":true}`)
	if f.shutdowns != 0 {
		t.Error("shut down on a check")
	}
	if len(got) != 1 || got[0] != (reply{ID: "1", Body: "ready"}) {
		t.Errorf("got %+v", got)
	}
}

func TestAnythingElseIsRefused(t *testing.T) {
	f := &fake{}
	got := f.serve(t,
		`{"crosstalk":{"hub":"connected"}}`,
		`not json`,
		`{"id":"1","from":"kiwi","body":"shutdown","wantsReply":true}`,
		`{"id":"2","from":"kiwi","body":{"do":"Shutdown"},"wantsReply":true}`,
		`{"id":"3","from":"kiwi","wantsReply":true}`,
		`{"id":"4","from":"kiwi","body":{"do":"shutdown","do":false},"wantsReply":true}`,
	)
	if f.shutdowns != 0 {
		t.Error("shut down")
	}
	if len(got) != 4 {
		t.Fatalf("got %+v, want 4 refusals", got)
	}
	for _, r := range got {
		if r.Error != "refused" {
			t.Errorf("got %+v", r)
		}
	}
}
