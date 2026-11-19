// circula-messages is an explicit HTTP client for the isolated messaging API.
// It starts no server and never opens a database or performs ledger operations.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

func main() {
	if e := run(os.Args[1:], os.Stdin, os.Stdout); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run(args []string, input io.Reader, output io.Writer) error {
	f := flag.NewFlagSet("circula-messages", flag.ContinueOnError)
	endpoint := f.String("url", "", "Circula HTTPS origin")
	network := f.String("network", "", "authorized network ID")
	thread := f.String("thread", "", "conversation ID")
	target := f.String("target", "", "member ID")
	reference := f.String("reference", "", "optional local listing/transaction ID")
	key := f.String("key", "", "stable idempotency key (8–128 ID characters)")
	cursor := f.String("cursor", "", "last conversation ID")
	reporter := f.String("reporter", "", "reporter ID for moderation or report paging")
	moderation := f.String("moderation-action", "", "hold, release, resolve or dismiss")
	after := f.Int64("after", 0, "last received sequence")
	days := f.Int("days", 30, "retention days, subject to network policy")
	if e := f.Parse(args); e != nil {
		return e
	}
	if f.NArg() != 1 {
		return errors.New("choose list, create, read, search, send, read-marker, hide, close, block, unblock, report, export, reports, held, moderate, purge or retention; flags precede command")
	}
	u, e := url.Parse(*endpoint)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" {
		return errors.New("url must be an HTTPS origin")
	}
	if u.Scheme != "https" {
		return errors.New("HTTPS required")
	}
	token := os.Getenv("CIRCULA_TOKEN")
	if token == "" || *network == "" {
		return errors.New("CIRCULA_TOKEN and -network required")
	}
	command := f.Arg(0)
	path := "/api/v1/conversations"
	method := "GET"
	var payload any
	switch command {
	case "list":
		if *cursor != "" {
			path += "?after=" + url.QueryEscape(*cursor)
		}
	case "create":
		method = "POST"
		payload = map[string]any{"target": *target, "reference": *reference, "retention_days": *days}
	case "read", "export":
		path += "/" + url.PathEscape(*thread) + "/messages"
		if command == "export" {
			path = strings.TrimSuffix(path, "messages") + "export"
		}
		path += "?after=" + fmt.Sprint(*after)
	case "search":
		b, err := io.ReadAll(io.LimitReader(input, 129))
		if err != nil {
			return err
		}
		if len(b) > 128 {
			return errors.New("stdin exceeds search limit")
		}
		if !utf8.Valid(b) {
			return errors.New("stdin is not valid UTF-8")
		}
		path += "/" + url.PathEscape(*thread) + "/search"
		method = "POST"
		payload = map[string]any{"query": string(b), "after": *after}
	case "send", "read-marker", "hide", "close", "report":
		action := command
		if action == "send" {
			action = "messages"
		}
		if action == "read-marker" {
			action = "read"
		}
		path += "/" + url.PathEscape(*thread) + "/" + action
		method = "POST"
		payload = struct{}{}
		if command == "read-marker" {
			payload = map[string]int64{"sequence": *after}
		}
		if command == "send" || command == "report" {
			limit := int64(4096)
			if command == "report" {
				limit = 1024
			}
			b, e := io.ReadAll(io.LimitReader(input, limit+1))
			if e != nil {
				return e
			}
			if int64(len(b)) > limit {
				return errors.New("stdin exceeds content limit")
			}
			if !utf8.Valid(b) {
				return errors.New("stdin is not valid UTF-8")
			}
			if command == "send" {
				payload = map[string]string{"body": string(b)}
			} else {
				payload = map[string]any{"sequence": *after, "reason": string(b)}
			}
		}
	case "reports", "held":
		path += "/" + url.PathEscape(*thread) + "/" + command + "?after=" + fmt.Sprint(*after) + "&reporter=" + url.QueryEscape(*reporter)
	case "moderate":
		path += "/" + url.PathEscape(*thread) + "/moderate"
		method = "POST"
		payload = map[string]any{"action": *moderation, "sequence": *after, "reporter": *reporter}
	case "purge":
		path = "/api/v1/purge"
		method = "POST"
		payload = struct{}{}
	case "retention":
		path = "/api/v1/retention"
		method = "POST"
		payload = map[string]int{"days": *days}
	case "block", "unblock":
		method = "POST"
		path = "/api/v1/block"
		payload = map[string]any{"target": *target, "enabled": command == "block"}
	default:
		return errors.New("unknown command")
	}
	var body bytes.Buffer
	if payload != nil {
		if e = json.NewEncoder(&body).Encode(payload); e != nil {
			return e
		}
	}
	req, e := http.NewRequest(method, strings.TrimSuffix(*endpoint, "/")+path, &body)
	if e != nil {
		return e
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Network", *network)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", *key)
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, e := client.Do(req)
	if e != nil {
		return errors.New("messaging connection failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("messaging request returned HTTP %d", response.StatusCode)
	}
	// Every response is bounded. Never follow redirects with credentials.
	const maxResponse = 4 << 20
	data, e := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if e != nil {
		return errors.New("messaging response read failed")
	}
	if len(data) > maxResponse {
		return errors.New("messaging response exceeds limit")
	}
	_, e = output.Write(data)
	return e
}
