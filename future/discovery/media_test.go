package discovery_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
	"github.com/djshellshoxxx/circula/future/discovery/internal/testkit"
)

func (e *env) upload(a d.Actor, listing string, data []byte) (d.MediaInfo, error) {
	return e.svc.AddMedia(ctx, a, listing, "A photo of the item", bytes.NewReader(data))
}

func TestUploadSanitizesAndServes(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bike", "")
	src := testkit.WithGPSEXIF(testkit.JPEG(t, 900, 300, 1), 1)
	if !bytes.Contains(src, []byte("GPSLatitude")) {
		t.Fatal("fixture")
	}
	m, err := e.upload(alice, "L1", src)
	if err != nil {
		t.Fatal(err)
	}
	if m.Width != 600 || m.Height != 200 || m.ThumbWidth != 120 || m.Status != d.MediaActive {
		t.Fatalf("%+v", m)
	}
	if strings.ContainsAny(m.StoragePath, " \\") || strings.Contains(m.StoragePath, "..") || strings.Contains(m.StoragePath, "alice") || strings.Contains(m.StoragePath, "L1") {
		t.Fatal("storage path leaks identifiers:", m.StoragePath)
	}
	for _, thumb := range []bool{false, true} {
		b, err := e.svc.OpenMedia(ctx, bob, "L1", m.ID, thumb)
		if err != nil {
			t.Fatal(err)
		}
		if b.MIME != "image/jpeg" || bytes.Contains(b.Data, []byte("GPS")) || bytes.Contains(b.Data, []byte("Exif")) || bytes.Contains(b.Data, []byte("HOME")) {
			t.Fatal("metadata survived or wrong mime")
		}
	}
	v, _ := e.svc.GetListing(ctx, bob, "L1")
	if len(v.Media) != 1 || v.Media[0].Alt == "" {
		t.Fatal(v.Media)
	}
	r := e.mustSearch(bob, d.Filters{})
	if r.Items[0].Cover == nil || r.Items[0].Cover.ID != m.ID {
		t.Fatal("no cover in results")
	}
	// png alpha path
	if m2, err := e.upload(alice, "L1", testkit.PNG(t, 40, 30)); err != nil || m2.MIME != "image/png" {
		t.Fatal(err, m2.MIME)
	}
}

func TestUploadRejections(t *testing.T) {
	e := newEnv(t)
	e.img.MaxInputBytes = 20000
	e.img.MaxPixels = 100000
	e.open()
	e.listing("n1", "L1", "alice", "Bike", "")
	e.listing("n1", "L2", "bob", "Saw", "")
	good := testkit.JPEG(t, 50, 50, 3)
	cases := map[string]struct {
		a    d.Actor
		id   string
		data []byte
		alt  string
		want error
	}{
		"svg":        {alice, "L1", []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`), "x", d.ErrInvalid},
		"zip":        {alice, "L1", append([]byte("PK\x03\x04"), make([]byte, 200)...), "x", d.ErrInvalid},
		"polyglot":   {alice, "L1", append(append([]byte{}, good...), []byte("PK\x03\x04<script>")...), "x", d.ErrInvalid},
		"huge dims":  {alice, "L1", testkit.JPEG(t, 400, 400, 1), "x", d.ErrQuota},
		"too big":    {alice, "L1", make([]byte, 30000), "x", d.ErrQuota},
		"empty":      {alice, "L1", nil, "x", d.ErrInvalid},
		"no alt":     {alice, "L1", good, "  ", d.ErrInvalid},
		"alt markup": {alice, "L1", good, "<b>x</b>", d.ErrInvalid},
		"not owner":  {bob, "L1", good, "x", d.ErrForbidden},
		"foreign":    {carol, "L1", good, "x", d.ErrNotFound},
		"missing":    {alice, "nope", good, "x", d.ErrNotFound},
		"inactive":   {d.Actor{ID: "alice", Network: "n1"}, "L1", good, "x", d.ErrForbidden},
		"admin":      {admin, "L1", good, "x", d.ErrForbidden}, // admins moderate, they do not upload for members
	}
	for name, c := range cases {
		_, err := e.svc.AddMedia(ctx, c.a, c.id, c.alt, bytes.NewReader(c.data))
		if !errors.Is(err, c.want) {
			t.Errorf("%s: got %v want %v", name, err, c.want)
		}
	}
	if v, _ := e.svc.GetListing(ctx, alice, "L1"); len(v.Media) != 0 {
		t.Fatal("rejected upload left media")
	}
	// nothing leaked to the blob store either
	if n := countFiles(t, e.dir+"/blobs"); n != 0 {
		t.Fatalf("%d blobs from rejected uploads", n)
	}
}

func TestSixPhotoLimitSequentialAndConcurrent(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bike", "")
	e.listing("n1", "L2", "alice", "Saw", "")
	for i := 0; i < 6; i++ {
		if _, err := e.upload(alice, "L1", testkit.JPEG(t, 80, 60, i)); err != nil {
			t.Fatal(i, err)
		}
	}
	if _, err := e.upload(alice, "L1", testkit.JPEG(t, 80, 60, 99)); !errors.Is(err, d.ErrQuota) {
		t.Fatal("seventh photo accepted:", err)
	}
	if _, err := e.upload(alice, "L1", testkit.JPEG(t, 80, 60, 0)); !errors.Is(err, d.ErrQuota) && !errors.Is(err, d.ErrDuplicate) {
		t.Fatal(err)
	}
	// concurrent uploads on a fresh listing never exceed six
	var wg sync.WaitGroup
	var mu sync.Mutex
	okN, quotaN := 0, 0
	for i := 0; i < 14; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := e.upload(alice, "L2", testkit.JPEG(t, 70, 70, 100+i))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				okN++
			case errors.Is(err, d.ErrQuota):
				quotaN++
			default:
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	if okN != 6 || quotaN != 8 {
		t.Fatalf("concurrent: ok=%d quota=%d", okN, quotaN)
	}
	v, _ := e.svc.GetListing(ctx, alice, "L2")
	if len(v.Media) != 6 {
		t.Fatal(len(v.Media))
	}
	// the cap is also bounded by config
	e.cfg.MaxPhotos = 99
	e.open()
	if e.svc.Config().MaxPhotos != 6 {
		t.Fatal("configurable limit may not exceed the spec ceiling of six")
	}
}

func TestListingByteQuotaAndDuplicate(t *testing.T) {
	e := newEnv(t)
	e.cfg.MaxListingBytes = 12000
	e.open()
	e.listing("n1", "L1", "alice", "Bike", "")
	first := testkit.JPEG(t, 100, 100, 1)
	_, err := e.upload(alice, "L1", first)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.upload(alice, "L1", first); !errors.Is(err, d.ErrDuplicate) {
		t.Fatal("duplicate accepted:", err)
	}
	var lastErr error
	for i := 2; i < 8 && lastErr == nil; i++ {
		_, lastErr = e.upload(alice, "L1", testkit.JPEG(t, 200, 200, i))
	}
	if !errors.Is(lastErr, d.ErrQuota) {
		t.Fatal("byte quota not enforced:", lastErr)
	}
}

func TestUploadRateLimit(t *testing.T) {
	e := newEnv(t)
	e.cfg.UploadsPerHour = 2
	e.open()
	e.listing("n1", "L1", "alice", "Bike", "")
	for i := 0; i < 2; i++ {
		if _, err := e.upload(alice, "L1", testkit.JPEG(t, 40, 40, i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := e.upload(alice, "L1", testkit.JPEG(t, 40, 40, 5)); !errors.Is(err, d.ErrRateLimited) {
		t.Fatal(err)
	}
	e.clock.Advance(61 * time.Minute)
	if _, err := e.upload(alice, "L1", testkit.JPEG(t, 40, 40, 6)); err != nil {
		t.Fatal(err)
	}
}

func TestMediaAuthorization(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bike", "")
	m, _ := e.upload(alice, "L1", testkit.JPEG(t, 60, 60, 1))
	if err := e.svc.RemoveMedia(ctx, bob, "L1", m.ID); !errors.Is(err, d.ErrForbidden) {
		t.Fatal("unauthorized delete:", err)
	}
	if err := e.svc.RemoveMedia(ctx, carol, "L1", m.ID); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("cross-network delete:", err)
	}
	if err := e.svc.RemoveMedia(ctx, admin2, "L1", m.ID); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("foreign admin delete:", err)
	}
	if _, err := e.svc.OpenMedia(ctx, carol, "L1", m.ID, false); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("cross-network read:", err)
	}
	// media id under the wrong listing is not found
	e.listing("n1", "L2", "bob", "Saw", "")
	if _, err := e.svc.OpenMedia(ctx, bob, "L2", m.ID, false); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("media readable via other listing:", err)
	}
	if _, err := e.svc.OpenMedia(ctx, bob, "L1", "../../etc/passwd", false); !errors.Is(err, d.ErrNotFound) {
		t.Fatal(err)
	}
	// admin may remove; removal is immediate
	if err := e.svc.RemoveMedia(ctx, admin, "L1", m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.OpenMedia(ctx, bob, "L1", m.ID, false); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("removed media still served:", err)
	}
	if _, err := e.svc.OpenMedia(ctx, alice, "L1", m.ID, true); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("removed media served to owner:", err)
	}
	if err := e.svc.RemoveMedia(ctx, alice, "L1", m.ID); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("double delete:", err)
	}
	// permissions re-evaluated on every read: owner suspended -> hidden
	m2, _ := e.upload(alice, "L1", testkit.JPEG(t, 60, 60, 2))
	if _, err := e.svc.OpenMedia(ctx, bob, "L1", m2.ID, false); err != nil {
		t.Fatal(err)
	}
	rec := e.listing("n1", "L1", "alice", "Bike", "")
	rec.OwnerActive = false
	e.svc.SyncListing(ctx, rec)
	if _, err := e.svc.OpenMedia(ctx, bob, "L1", m2.ID, false); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("media served after owner suspension:", err)
	}
	if _, err := e.svc.OpenMedia(ctx, d.Actor{ID: "bob", Network: "n1"}, "L1", m2.ID, false); !errors.Is(err, d.ErrForbidden) {
		t.Fatal(err)
	}
}

func TestExpiryAwareMediaReads(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bike", "")
	m, _ := e.upload(alice, "L1", testkit.JPEG(t, 60, 60, 1))
	if _, err := e.svc.UpdateDiscovery(ctx, alice, "L1", d.DiscoveryInput{ExpiresAt: e.clock.Now().Add(24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.OpenMedia(ctx, bob, "L1", m.ID, true); err != nil {
		t.Fatal(err)
	}
	e.clock.Advance(24 * time.Hour)
	if _, err := e.svc.OpenMedia(ctx, bob, "L1", m.ID, true); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("expired listing media served to member:", err)
	}
	if _, err := e.svc.OpenMedia(ctx, alice, "L1", m.ID, true); err != nil {
		t.Fatal("owner must still see own expired media:", err)
	}
	if _, err := e.svc.Renew(ctx, alice, "L1", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.OpenMedia(ctx, bob, "L1", m.ID, true); err != nil {
		t.Fatal("renewed listing media should serve again:", err)
	}
}

func TestRetentionRemovedAndArchived(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bike", "")
	e.listing("n1", "L2", "alice", "Saw", "")
	m1, _ := e.upload(alice, "L1", testkit.JPEG(t, 60, 60, 1))
	m2, _ := e.upload(alice, "L2", testkit.JPEG(t, 60, 60, 2))
	e.svc.RemoveMedia(ctx, alice, "L1", m1.ID)
	// L2 is archived by the host adapter
	rec := e.listing("n1", "L2", "alice", "Saw", "")
	rec.Status = d.StatusArchived
	rec.Updated = e.clock.Now()
	e.svc.SyncListing(ctx, rec)
	// hidden immediately for members; still stored
	if _, err := e.svc.OpenMedia(ctx, bob, "L2", m2.ID, false); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("archived media visible:", err)
	}
	if _, err := e.svc.OpenMedia(ctx, alice, "L2", m2.ID, false); err != nil {
		t.Fatal("owner should still see archived media before GC:", err)
	}
	rep, err := e.svc.RunRetention(ctx)
	if err != nil || rep.Claimed != 0 {
		t.Fatal("GC before retention:", rep, err)
	}
	e.clock.Advance(30*24*time.Hour - time.Second)
	if rep, _ := e.svc.RunRetention(ctx); rep.Claimed != 0 {
		t.Fatal("GC one second early:", rep)
	}
	e.clock.Advance(time.Second)
	rep, err = e.svc.RunRetention(ctx)
	if err != nil || rep.Claimed != 2 || rep.RowsDeleted != 2 || rep.BlobsDeleted != 4 || len(rep.Errors) != 0 {
		t.Fatalf("%+v %v", rep, err)
	}
	if n := countFiles(t, e.dir+"/blobs"); n != 0 {
		t.Fatalf("%d blobs left after GC", n)
	}
	if _, err := e.store.GetMedia(ctx, "n1", "L1", m1.ID); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("row survived GC")
	}
	// idempotent
	if rep, _ := e.svc.RunRetention(ctx); rep.Claimed != 0 {
		t.Fatal(rep)
	}
	// slots are free again
	if _, err := e.upload(alice, "L1", testkit.JPEG(t, 60, 60, 7)); err != nil {
		t.Fatal(err)
	}
}

func TestUnarchivedBeforeRetentionKeepsMedia(t *testing.T) {
	e := newEnv(t)
	rec := e.listing("n1", "L1", "alice", "Bike", "")
	m, _ := e.upload(alice, "L1", testkit.JPEG(t, 60, 60, 1))
	rec.Status = d.StatusArchived
	e.svc.SyncListing(ctx, rec)
	e.clock.Advance(10 * 24 * time.Hour)
	rec.Status, rec.Updated = d.StatusActive, e.clock.Now()
	e.svc.SyncListing(ctx, rec)
	e.clock.Advance(40 * 24 * time.Hour)
	if rep, _ := e.svc.RunRetention(ctx); rep.Claimed != 0 {
		t.Fatalf("restored listing lost media: %+v", rep)
	}
	if _, err := e.svc.OpenMedia(ctx, alice, "L1", m.ID, false); err != nil {
		t.Fatal(err)
	}
}

func TestCrashBetweenMetadataAndBlobs(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bike", "")
	mem := d.NewMemBlobs()
	svc, _ := d.NewService(e.store, mem, e.cfg, e.img)
	mem.FailPut = func(key string) error {
		if strings.Contains(key, ".thumb.") {
			return errors.New("disk full")
		}
		return nil
	}
	if _, err := svc.AddMedia(ctx, alice, "L1", "alt", bytes.NewReader(testkit.JPEG(t, 60, 60, 1))); err == nil {
		t.Fatal("expected failure")
	}
	if len(mem.Keys()) != 0 {
		t.Fatal("orphan blobs after failed upload:", mem.Keys())
	}
	if v, _ := svc.GetListing(ctx, alice, "L1"); len(v.Media) != 0 {
		t.Fatal("half-written media visible")
	}
	// process dies after metadata reserve but before anything else: simulate
	// by reserving directly; the abandoned pending row is invisible, counts
	// against quota until the pending TTL, then is reclaimed.
	pend := d.MediaInfo{ID: "med_pending", Network: "n1", ListingID: "L1", OwnerID: "alice", SHA256: "abc", MIME: "image/jpeg", Alt: "a",
		Width: 1, Height: 1, ThumbWidth: 1, ThumbHeight: 1, Bytes: 10, ThumbBytes: 10, StoragePath: "media/x/p.jpg", ThumbPath: "media/x/p.thumb.jpg",
		Status: d.MediaPending, Created: e.clock.Now()}
	if err := e.store.ReserveMedia(ctx, pend, d.MediaQuota{MaxCount: 6, MaxBytes: 1 << 20}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.OpenMedia(ctx, alice, "L1", "med_pending", false); !errors.Is(err, d.ErrNotFound) {
		t.Fatal("pending media served:", err)
	}
	// the failed upload above was handed to GC immediately; the abandoned
	// pending row is not reclaimed until its TTL passes.
	if rep, _ := svc.RunRetention(ctx); rep.Claimed != 1 || rep.RowsDeleted != 1 {
		t.Fatal("failed-upload row not reclaimed", rep)
	}
	if rep, _ := svc.RunRetention(ctx); rep.Claimed != 0 {
		t.Fatal("pending reclaimed before TTL", rep)
	}
	e.clock.Advance(61 * time.Minute)
	rep, err := svc.RunRetention(ctx)
	if err != nil || rep.RowsDeleted != 1 {
		t.Fatalf("%+v %v", rep, err)
	}
}

func TestRestartDurability(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bike", "")
	e.svc.CreateCategory(ctx, admin, "Tools", d.TypeGoods)
	m, err := e.upload(alice, "L1", testkit.JPEG(t, 90, 60, 1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.UpdateDiscovery(ctx, alice, "L1", d.DiscoveryInput{Tags: []string{"red"}, ExpiresAt: e.clock.Now().Add(72 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	r, _ := e.svc.Search(ctx, bob, d.Filters{Limit: 1}, "")
	_ = r
	before, _ := e.svc.OpenMedia(ctx, bob, "L1", m.ID, false)
	e.open() // close and reopen database + blob store + service
	after, err := e.svc.OpenMedia(ctx, bob, "L1", m.ID, false)
	if err != nil || !bytes.Equal(before.Data, after.Data) {
		t.Fatal("media lost across restart:", err)
	}
	v, err := e.svc.GetListing(ctx, alice, "L1")
	if err != nil || len(v.Tags) != 1 || v.ExpiresAt.IsZero() || v.Version != 1 {
		t.Fatalf("metadata lost across restart: %+v %v", v, err)
	}
	if cats, _ := e.svc.ListCategories(ctx, admin); len(cats) != 1 {
		t.Fatal("categories lost")
	}
}

func TestIntegrityCheckOnTamperedBlob(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bike", "")
	m, _ := e.upload(alice, "L1", testkit.JPEG(t, 60, 60, 1))
	if err := e.blobs.Put(ctx, m.StoragePath, []byte("replaced by attacker")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.OpenMedia(ctx, bob, "L1", m.ID, false); err == nil || errors.Is(err, d.ErrNotFound) {
		t.Fatal("tampered blob served:", err)
	}
}

func TestFSBlobsRejectTraversal(t *testing.T) {
	b, err := d.NewFSBlobs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"", "../x", "a/../../x", "/abs", "a//b", "a\\b", "a/.", "a b", "a\x00b", strings.Repeat("a", 300)} {
		if err := b.Put(ctx, k, []byte("x")); err == nil {
			t.Errorf("key %q accepted", k)
		}
	}
	if err := b.Put(ctx, "media/ab12/x.jpg", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Get(ctx, "media/ab12/missing.jpg"); !errors.Is(err, d.ErrNotFound) {
		t.Fatal(err)
	}
	if err := b.Delete(ctx, "media/ab12/missing.jpg"); err != nil {
		t.Fatal(err)
	}
}

func TestAuditTrail(t *testing.T) {
	e := newEnv(t)
	e.listing("n1", "L1", "alice", "Bike", "")
	e.svc.CreateCategory(ctx, admin, "Tools", d.TypeGoods)
	m, _ := e.upload(alice, "L1", testkit.JPEG(t, 60, 60, 1))
	e.svc.RemoveMedia(ctx, alice, "L1", m.ID)
	e.svc.ModerateTag(ctx, admin, "okay", d.TagApproved)
	entries, err := e.svc.Audit(ctx, admin, 50)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, en := range entries {
		got[en.Action] = true
	}
	for _, want := range []string{"category.create", "media.add", "media.remove", "tag.moderate"} {
		if !got[want] {
			t.Errorf("missing audit %s in %v", want, fmt.Sprint(got))
		}
	}
	if _, err := e.svc.Audit(ctx, alice, 10); !errors.Is(err, d.ErrForbidden) {
		t.Fatal(err)
	}
	if en, _ := e.svc.Audit(ctx, admin2, 10); len(en) != 0 {
		t.Fatal("audit leaked across networks")
	}
}
