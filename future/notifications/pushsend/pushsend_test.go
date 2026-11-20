package pushsend

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/future/notifications"
	"github.com/djshellshoxxx/circula/future/notifications/notifytest"
)

type transportClientFunc func(*http.Request) (*http.Response, error)

func (f transportClientFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func transportConfig(t *testing.T, client transportClientFunc) Config {
	t.Helper()
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return Config{
		VAPIDPublic:  base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()),
		VAPIDPrivate: notifications.NewSecret(base64.RawURLEncoding.EncodeToString(key.Bytes())),
		Subject:      "mailto:operator@example.test",
		Policy:       notifications.EndpointPolicy{AllowedHosts: []string{"push.example.test"}},
		Client:       client,
	}
}

func transportMessage() notifications.PushMessage {
	sub := notifytest.NewSubscriber()
	return notifications.PushMessage{
		Endpoint: "https://push.example.test/subscription-canary",
		P256dh:   sub.P256dh(), Auth: sub.AuthB64(),
		Payload: []byte(`{"title":"Circula","path":"/notifications"}`),
		TTL:     5 * time.Minute, Topic: notifications.PushTag,
	}
}

func TestSendPushSuccessReturnsNilError(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusNoContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			cfg := transportConfig(t, func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
			})
			sender, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if err := sender.SendPush(context.Background(), transportMessage()); err != nil {
				t.Fatalf("successful HTTP %d returned a nonnil error (%T)", status, err)
			}
			if calls != 1 {
				t.Fatalf("sent %d requests, want exactly one", calls)
			}
		})
	}
}

func TestRetryAfterCannotOverflow(t *testing.T) {
	for _, value := range []string{"9223372036854775807", "18446744073709551615", "99999999999999999999999999999999"} {
		t.Run(value, func(t *testing.T) {
			resp := &http.Response{Header: http.Header{"Retry-After": []string{value}}}
			if got := retryAfter(resp); got != notifications.MaxRetryAfter {
				t.Fatalf("Retry-After %q: got %s, want capped %s", value, got, notifications.MaxRetryAfter)
			}
		})
	}
}

func TestSendPushUsesEncryptedPolicyCheckedTransport(t *testing.T) {
	pki := notifytest.NewPKI()
	server := notifytest.StartPush(pki.Leaf("push.example.test"))
	t.Cleanup(server.Close)
	sub := notifytest.NewSubscriber()
	server.Register("/subscription-canary", sub)
	dialer := &notifytest.RedirectDialer{Target: server.Listener.Addr().String()}
	policy := notifications.EndpointPolicy{AllowedHosts: []string{"push.example.test"}}
	client := (notifications.Egress{
		Policy: policy, RootCAs: pki.Pool, Dialer: dialer, Timeout: 2 * time.Second,
		Resolver: notifytest.StaticResolver{"push.example.test": []netip.Addr{netip.MustParseAddr("8.8.8.8")}},
	}).Client()
	t.Cleanup(client.CloseIdleConnections)
	cfg := transportConfig(t, client.Do)
	sender, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	message := transportMessage()
	message.P256dh, message.Auth = sub.P256dh(), sub.AuthB64()
	if err := sender.SendPush(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	received := server.Requests()
	if len(received) != 1 || !bytes.Equal(received[0].Plain, message.Payload) {
		t.Fatal("loopback provider could not decrypt the exact payload")
	}
	request := received[0]
	if bytes.Contains(request.Body, message.Payload) || request.Header.Get("Content-Encoding") != "aes128gcm" {
		t.Fatal("provider received plaintext or unexpected content encoding")
	}
	if request.Header.Get("TTL") != "300" || request.Header.Get("Topic") != notifications.PushTag || !strings.HasPrefix(request.Header.Get("Authorization"), "vapid t=") {
		t.Fatal("TTL, topic or VAPID authentication was not preserved")
	}
	if got := dialer.Dialed(); len(got) != 1 || got[0] != "8.8.8.8:443" {
		t.Fatalf("egress did not validate and dial the resolved public address: %v", got)
	}
	// The test dialer connects only to the loopback fixture. A provider's
	// redirect must not result in another request, even to an allowed host.
	server.ReplyWith("/subscription-canary", func(w http.ResponseWriter) {
		w.Header().Set("Location", "https://push.example.test/elsewhere-canary")
		w.WriteHeader(http.StatusFound)
	})
	err = sender.SendPush(context.Background(), message)
	assertPushClass(t, err, notifications.ClassPermanent, "redirect_refused")
	if got := server.Requests(); len(got) != 2 || got[1].Path != "/subscription-canary" {
		t.Fatal("provider redirect was followed")
	}
}

type trackedPushBody struct {
	reader io.Reader
	read   int
	closed bool
}

func (b *trackedPushBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	b.read += n
	return n, err
}

func (b *trackedPushBody) Close() error { b.closed = true; return nil }

func TestSendPushClassifiesStatusesAndBoundsBody(t *testing.T) {
	for _, tc := range []struct {
		status int
		class  notifications.SendClass
		code   string
	}{
		{http.StatusCreated, notifications.ClassTransient, ""},
		{http.StatusNotFound, notifications.ClassTargetGone, "subscription_gone"},
		{http.StatusGone, notifications.ClassTargetGone, "subscription_gone"},
		{http.StatusTooManyRequests, notifications.ClassRateLimited, "rate_limited"},
		{http.StatusRequestEntityTooLarge, notifications.ClassPermanent, "payload_too_large"},
		{http.StatusFound, notifications.ClassPermanent, "redirect_refused"},
		{http.StatusUnauthorized, notifications.ClassPermanent, "vapid_rejected"},
		{http.StatusForbidden, notifications.ClassPermanent, "vapid_rejected"},
		{http.StatusServiceUnavailable, notifications.ClassTransient, "provider_5xx"},
		{http.StatusBadRequest, notifications.ClassPermanent, "provider_4xx"},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			body := &trackedPushBody{reader: strings.NewReader(strings.Repeat("provider-secret-canary ", 512))}
			cfg := transportConfig(t, func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Header: http.Header{"Retry-After": []string{"900"}}, Body: body}, nil
			})
			sender, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			err = sender.SendPush(context.Background(), transportMessage())
			if tc.code == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assertPushClass(t, err, tc.class, tc.code)
				if tc.status == http.StatusTooManyRequests || tc.status == http.StatusServiceUnavailable {
					var sendErr *notifications.SendError
					if !errors.As(err, &sendErr) || sendErr.RetryAfter != 15*time.Minute {
						t.Fatal("provider retry delay was not preserved")
					}
				}
			}
			if !body.closed || body.read > 2048 {
				t.Fatalf("response body was not bounded/closed: read=%d closed=%t", body.read, body.closed)
			}
		})
	}
}

func TestSendPushRejectsInvalidInputsBeforeRequest(t *testing.T) {
	for _, tc := range []struct {
		name  string
		edit  func(*notifications.PushMessage)
		class notifications.SendClass
		code  string
	}{
		{"empty payload", func(m *notifications.PushMessage) { m.Payload = nil }, notifications.ClassPermanent, "payload_size"},
		{"oversize payload", func(m *notifications.PushMessage) { m.Payload = make([]byte, MaxPayload+1) }, notifications.ClassPermanent, "payload_size"},
		{"plaintext endpoint", func(m *notifications.PushMessage) { m.Endpoint = "http://push.example.test/token-canary" }, notifications.ClassTargetGone, "endpoint_policy"},
		{"unlisted endpoint", func(m *notifications.PushMessage) { m.Endpoint = "https://unlisted.example.test/token-canary" }, notifications.ClassTargetGone, "endpoint_policy"},
		{"invalid recipient key", func(m *notifications.PushMessage) { m.P256dh = "key-canary!" }, notifications.ClassPermanent, "encrypt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			cfg := transportConfig(t, func(*http.Request) (*http.Response, error) {
				calls++
				return nil, errors.New("must not request")
			})
			sender, err := New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			message := transportMessage()
			tc.edit(&message)
			assertPushClass(t, sender.SendPush(context.Background(), message), tc.class, tc.code)
			if calls != 0 {
				t.Fatal("invalid input reached the HTTP client")
			}
		})
	}
}

func TestSendPushBlocksPrivateDNSBeforeDial(t *testing.T) {
	dialer := &notifytest.RedirectDialer{Target: "127.0.0.1:1"}
	client := (notifications.Egress{
		Policy: notifications.EndpointPolicy{AllowedHosts: []string{"push.example.test"}}, Dialer: dialer,
		Resolver: notifytest.StaticResolver{"push.example.test": []netip.Addr{netip.MustParseAddr("127.0.0.1")}},
	}).Client()
	t.Cleanup(client.CloseIdleConnections)
	sender, err := New(transportConfig(t, client.Do))
	if err != nil {
		t.Fatal(err)
	}
	assertPushClass(t, sender.SendPush(context.Background(), transportMessage()), notifications.ClassPermanent, "egress_blocked")
	if len(dialer.Dialed()) != 0 {
		t.Fatal("private DNS answer reached the socket dialer")
	}
}

func TestSendPushPreservesCancellation(t *testing.T) {
	cfg := transportConfig(t, func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })
	sender, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = sender.SendPush(ctx, transportMessage())
	assertPushClass(t, err, notifications.ClassTransient, "canceled")
	if !errors.Is(err, context.Canceled) {
		t.Fatal("send error lost context cancellation")
	}
}

func TestPushConfigurationRequiresSafeClientAndMatchingKeys(t *testing.T) {
	cfg := transportConfig(t, func(*http.Request) (*http.Response, error) { return nil, errors.New("unused") })
	missingClient := cfg
	missingClient.Client = nil
	if _, err := New(missingClient); err == nil {
		t.Fatal("missing client would allow upstream's unconstrained default")
	}
	wrongKey := transportConfig(t, nil)
	wrongKey.Client = cfg.Client
	wrongKey.VAPIDPublic = cfg.VAPIDPublic
	if _, err := New(wrongKey); err == nil {
		t.Fatal("mismatched VAPID key pair accepted")
	}
}

func assertPushClass(t *testing.T, err error, class notifications.SendClass, code string) {
	t.Helper()
	var sendErr *notifications.SendError
	if !errors.As(err, &sendErr) || sendErr == nil || sendErr.Class != class || sendErr.Code != code {
		t.Fatalf("error=%v (%T), want %s/%s", err, err, class, code)
	}
	for _, secret := range []string{"subscription-canary", "provider-secret-canary", "token-canary", "key-canary"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("provider error leaked %s", secret)
		}
	}
}

func TestRetryAfterKnownValues(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  time.Duration
	}{
		{"", 0}, {"0", 0}, {"15", 15 * time.Second}, {" 15 ", 15 * time.Second},
		{"0001", time.Second}, {"3600", notifications.MaxRetryAfter}, {"3601", notifications.MaxRetryAfter},
		{"-1", 0}, {"+1", 0}, {"invalid", 0}, {"999999999999999999999999999999invalid", 0},
		{time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat), 0},
		{time.Now().Add(2 * notifications.MaxRetryAfter).UTC().Format(http.TimeFormat), notifications.MaxRetryAfter},
	} {
		t.Run(tc.value, func(t *testing.T) {
			if got := retryAfter(&http.Response{Header: http.Header{"Retry-After": []string{tc.value}}}); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func FuzzRetryAfter(f *testing.F) {
	for _, value := range []string{"0", "900", "9223372036854775807", "99999999999999999999999999999999", "Wed, 01 Jan 2031 00:00:00 GMT", "-1"} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > 16<<10 {
			return // production response headers are bounded by the egress client
		}
		got := retryAfter(&http.Response{Header: http.Header{"Retry-After": []string{value}}})
		if got < 0 || got > notifications.MaxRetryAfter {
			t.Fatalf("retry delay escaped its bounds: %s", got)
		}
	})
}
