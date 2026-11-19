package recurring

import (
	"testing"
	"time"
)

func validTemplate() Template {
	return Template{
		Network: "network", ID: "template", Owner: "owner", Target: "target",
		Timezone: "UTC", Hour: 12, Minute: 0, Active: true, MembersActive: true,
	}
}

func TestDueRejectsInvalidIdentityEncoding(t *testing.T) {
	for _, field := range []string{"network", "id", "owner", "target"} {
		t.Run(field, func(t *testing.T) {
			x := validTemplate()
			switch field {
			case "network":
				x.Network = "network\xff"
			case "id":
				x.ID = "template\xff"
			case "owner":
				x.Owner = "owner\xff"
			case "target":
				x.Target = "target\xff"
			}
			if _, ok, err := Due(x, x.Network, at("2024-01-01T12:00:00Z")); err == nil || ok {
				t.Fatalf("invalid %s encoding accepted: due=%v err=%v", field, ok, err)
			}
		})
	}
}

func TestDueRejectsHostLocalTimezone(t *testing.T) {
	x := validTemplate()
	x.Timezone = "Local"
	if _, ok, err := Due(x, x.Network, at("2024-01-01T12:00:00Z")); err == nil || ok {
		t.Fatalf("host-dependent timezone accepted: due=%v err=%v", ok, err)
	}
}

func TestDueHistoricalOffsetSeconds(t *testing.T) {
	x := validTemplate()
	x.Timezone = "Europe/Paris"
	when := at("1900-01-01T11:50:39Z") // Paris mean time was UTC+00:09:21.
	if _, ok, err := Due(x, x.Network, when.Add(-time.Nanosecond)); err != nil || ok {
		t.Fatalf("early reminder: due=%v err=%v", ok, err)
	}
	occurrence, ok, err := Due(x, x.Network, when)
	if err != nil || !ok || !occurrence.Scheduled.Equal(when) {
		t.Fatalf("exact historical minute not selected: occurrence=%+v due=%v err=%v", occurrence, ok, err)
	}
}

func TestDueAtZeroInstant(t *testing.T) {
	x := validTemplate()
	x.Hour = 0
	occurrence, ok, err := Due(x, x.Network, time.Time{})
	if err != nil || !ok || !occurrence.Scheduled.Equal(time.Time{}) {
		t.Fatalf("valid zero instant confused with no occurrence: occurrence=%+v due=%v err=%v", occurrence, ok, err)
	}
}
