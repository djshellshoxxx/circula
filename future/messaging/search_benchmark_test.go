package messaging

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// Exercise both early-match and full-scan behavior with a large synthetic inbox.
// Timing is diagnostic and is not a production throughput guarantee.
func BenchmarkSearchTenThousandMessages(b *testing.B) {
	f := setup(b)
	thread := f.thread()
	tx, err := f.s.db.BeginTx(ctx, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer tx.Rollback()
	statement, err := tx.Prepare(`INSERT INTO message_entries(network,thread,sequence,id,sender,body,created) VALUES(?,?,?,?,?,?,?)`)
	if err != nil {
		b.Fatal(err)
	}
	text := strings.Repeat("private text ", 300)
	for i := 1; i <= 10000; i++ {
		if _, err = statement.Exec(alice.Network, thread.ID, i, fmt.Sprintf("benchmark-%d", i), alice.Member, text, f.now.UnixNano()); err != nil {
			b.Fatal(err)
		}
	}
	if err = statement.Close(); err != nil {
		b.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE message_threads SET sequence=10000 WHERE network=? AND id=?`, alice.Network, thread.ID); err != nil {
		b.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		b.Fatal(err)
	}
	for _, query := range []string{"private", "no-match"} {
		b.Run(query, func(b *testing.B) {
			want := 0
			if query == "private" {
				want = 100
			}
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				messages, err := f.s.Search(ctx, bob, thread.ID, query, 0, 100)
				if err != nil || len(messages) != want {
					b.Fatalf("search: count=%d want=%d error=%v", len(messages), want, err)
				}
			}
		})
	}
}

func TestSearchCancelledRequestReturnsNoContent(t *testing.T) {
	f := setup(t)
	thread := f.thread()
	f.send(thread.ID, "private content", "cancel-01")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	messages, err := f.s.Search(ctx, bob, thread.ID, "private", 0, 100)
	if err == nil || len(messages) != 0 {
		t.Fatalf("cancelled request released content: count=%d error=%v", len(messages), err)
	}
}
