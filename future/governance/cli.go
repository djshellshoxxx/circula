package governance

import (
	"context"
	"encoding/json"
	"errors"
	"io"
)

// RunCLI takes a host-authenticated identity and a JSON Command on input for
// mutation. No credentials are parsed or stored and no network server is opened.
func RunCLI(ctx context.Context, s Service, id Identity, args []string, input io.Reader, out io.Writer) error {
	if len(args) < 1 {
		return errors.New("usage: list | show ID | command")
	}
	switch args[0] {
	case "list":
		if len(args) != 1 && len(args) != 2 {
			return ErrDenied
		}
		after := ""
		if len(args) == 2 {
			after = args[1]
		}
		data, e := s.List(ctx, id, after, 100)
		if e != nil {
			return e
		}
		return json.NewEncoder(out).Encode(data)
	case "show":
		if len(args) != 2 {
			return ErrDenied
		}
		data, e := s.Read(ctx, id, args[1])
		if e != nil {
			return e
		}
		return json.NewEncoder(out).Encode(data)
	case "command":
		if len(args) != 1 || input == nil {
			return ErrDenied
		}
		raw, err := io.ReadAll(io.LimitReader(input, 65537))
		if err != nil {
			return err
		}
		if len(raw) > 65536 {
			return errors.New("command exceeds 64KiB")
		}
		var c Command
		if err := strictDecode(raw, &c); err != nil {
			return err
		}

		reply, e := s.Execute(ctx, id, c)
		if e != nil {
			return e
		}
		return json.NewEncoder(out).Encode(reply)
	default:
		return errors.New("unknown governance command")
	}
}
