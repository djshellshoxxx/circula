package recurring

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
)

// RunCLI accepts only a host-authenticated identity and already constructed
// service. It neither discovers credentials nor opens production databases.
// save input JSON is the same bounded Record used by HTTP; replay keys are
// deliberate user action IDs, not reused automatically across distinct actions.
func RunCLI(ctx context.Context, s Service, a Actor, args []string, in io.Reader, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: list [after] | save KEY EXPECTED | pause|resume|skip|cancel ID VERSION KEY | draft ID KEY | preview ID | inbox")
	}
	switch args[0] {
	case "list":
		if len(args) > 2 {
			return errors.New("list takes optional cursor")
		}
		after := ""
		if len(args) == 2 {
			after = args[1]
		}
		rows, next, err := s.List(ctx, a, after, 20)
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(struct {
			Rows []Record
			Next string
		}{rows, next})
	case "save":
		if len(args) != 3 || in == nil {
			return errors.New("save requires key revision and JSON stdin")
		}
		version, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return err
		}
		bounded := io.LimitReader(in, 65537)
		raw, err := io.ReadAll(bounded)
		if err != nil || len(raw) > 65536 {
			if err != nil {
				return err
			}
			return errors.New("input too large")
		}
		var r Record
		if err = decodeJSON(raw, &r); err != nil {
			return err
		}
		saved, err := s.Save(ctx, a, r, version, args[1])
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(saved)
	case "pause", "resume", "skip", "cancel":
		if len(args) != 4 {
			return errors.New("state action requires id revision key")
		}
		version, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
			return err
		}
		r, err := s.Change(ctx, a, args[1], args[0], version, args[3])
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(r)
	case "draft":
		if len(args) != 3 {
			return errors.New("draft requires id key")
		}
		draft, err := s.CreateDraft(ctx, a, args[1], args[2])
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(draft)
	case "inbox":
		if len(args) != 1 {
			return errors.New("unexpected arguments")
		}
		reminders, err := s.Inbox(ctx, a, 100)
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(reminders)
	case "preview":
		if len(args) != 2 {
			return errors.New("preview requires id")
		}
		r, err := s.Get(ctx, a, args[1])
		if err != nil {
			return err
		}
		window, ok, err := r.Calendar.Next(a.Network, r.ID, s.Now())
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(struct {
			Next     Occurrence
			Exists   bool
			Active   bool
			Advisory string
		}{window, ok, r.Active && !r.Cancelled, "Reminder only; no automatic trade"})
	default:
		return errors.New("unknown recurring command")
	}
}
