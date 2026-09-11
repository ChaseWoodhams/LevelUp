package wire

import (
	"context"
	"testing"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
)

type fakeMedalNamer struct {
	names  map[int64]medalNamePair
	called bool
	gotIDs []int64
}

func (f *fakeMedalNamer) MedalNames(_ context.Context, ids []int64) map[int64]medalNamePair {
	f.called = true
	f.gotIDs = ids
	out := make(map[int64]medalNamePair, len(ids))
	for _, id := range ids {
		if p, ok := f.names[id]; ok {
			out[id] = p
		}
	}
	return out
}

func medalSet(ids ...int64) map[int64]struct{} {
	m := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		m[id] = struct{}{}
	}
	return m
}

func TestEmitMedalFirstEarned_NewMedalEmitsNamedNotification(t *testing.T) {
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3)}
	namer := &fakeMedalNamer{names: map[int64]medalNamePair{3: {EN: "Perfect"}}}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "player-slug", before, after)

	if len(em.emitted) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(em.emitted))
	}
	in := em.emitted[0]
	if in.Category != notifications.CategoryMedalFirstEarned || in.Severity != notifications.SeveritySuccess {
		t.Fatalf("unexpected notification metadata: %+v", in)
	}
	if in.Params[paramKeyMedalName] != "Perfect" {
		t.Errorf("medal_name_en = %v, want Perfect", in.Params[paramKeyMedalName])
	}
	if in.TargetRoute != "/t/halo_infinite/players/player-slug/career/medals" || !targetRouteIsValid(in.TargetRoute) {
		t.Errorf("unexpected target route %q", in.TargetRoute)
	}
}

func TestEmitMedalFirstEarned_ColdStartSeedsSilently(t *testing.T) {
	before := &PlayerSnapshot{}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3, 4, 5)}
	namer := &fakeMedalNamer{}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "p", before, after)

	if len(em.emitted) != 0 || namer.called {
		t.Fatalf("cold start should emit nothing and skip name resolution")
	}
}

func TestEmitMedalFirstEarned_RecapAboveThreshold(t *testing.T) {
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3, 4, 5)}
	namer := &fakeMedalNamer{}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "p", before, after)

	if len(em.emitted) != 1 || em.emitted[0].TitleKey != "notif.medal_first_earned.recap.title" {
		t.Fatalf("expected one recap notification, got %+v", em.emitted)
	}
	if em.emitted[0].Params[paramKeyCount] != 4 || namer.called {
		t.Fatalf("unexpected recap payload or name lookup: %+v, called=%v", em.emitted[0].Params, namer.called)
	}
}

func TestEmitMedalFirstEarned_AtThresholdEmitsPerMedal(t *testing.T) {
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3, 4)}
	namer := &fakeMedalNamer{names: map[int64]medalNamePair{
		2: {EN: "A"}, 3: {EN: "B"}, 4: {EN: "C"},
	}}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "p", before, after)

	if n := countCategory(em.emitted, notifications.CategoryMedalFirstEarned); n != 3 {
		t.Fatalf("expected 3 notifications, got %d", n)
	}
}

func TestEmitMedalFirstEarned_NoChangeNoEmit(t *testing.T) {
	snap := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3)}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, &fakeMedalNamer{}, "halo_infinite", "p", snap, snap)

	if len(em.emitted) != 0 {
		t.Fatalf("expected no notification, got %d", len(em.emitted))
	}
}

func TestEmitMedalFirstEarned_UnresolvedNameSkipped(t *testing.T) {
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3)}
	namer := &fakeMedalNamer{names: map[int64]medalNamePair{2: {EN: "Resolved"}}}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "p", before, after)

	if len(em.emitted) != 1 || em.emitted[0].Params[paramKeyMedalName] != "Resolved" {
		t.Fatalf("expected only the named medal, got %+v", em.emitted)
	}
}

func TestEmitMedalFirstEarned_EmptyNameSkipped(t *testing.T) {
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2)}
	namer := &fakeMedalNamer{names: map[int64]medalNamePair{2: {}}}
	em := &recordingEmitter{}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "p", before, after)

	if len(em.emitted) != 0 {
		t.Fatalf("expected empty labels to be skipped, got %d", len(em.emitted))
	}
}

func TestEmitMedalFirstEarned_EmitErrorIsBestEffort(t *testing.T) {
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2, 3)}
	namer := &fakeMedalNamer{names: map[int64]medalNamePair{2: {EN: "A"}, 3: {EN: "B"}}}
	em := &recordingEmitter{failOn: notifications.CategoryMedalFirstEarned}

	emitMedalFirstEarned(context.Background(), em, namer, "halo_infinite", "p", before, after)

	if len(em.emitted) != 0 {
		t.Fatalf("expected no recorded notification, got %d", len(em.emitted))
	}
}

func TestNewMedalNamerForPDB_NilReturnsNil(t *testing.T) {
	if n := newMedalNamerForPDB(nil); n != nil {
		t.Error("nil pdb should return nil resolver")
	}
	if n := newMedalNamerForPDB(&duckdb.PlayerDB{}); n != nil {
		t.Error("pdb without metadata should return nil resolver")
	}
}

func TestEmitMedalFirstEarned_NilGuards(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{EarnedMedalIDs: medalSet(1)}
	after := &PlayerSnapshot{EarnedMedalIDs: medalSet(1, 2)}

	emitMedalFirstEarned(context.Background(), em, nil, "halo_infinite", "p", before, after)
	emitMedalFirstEarned(context.Background(), nil, &fakeMedalNamer{}, "halo_infinite", "p", before, after)

	if len(em.emitted) != 0 {
		t.Fatalf("nil guards should emit nothing, got %d", len(em.emitted))
	}
}
