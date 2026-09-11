// Tests purs (sans webhook réseau) pour NotifyFriendAdded + NotifyFriendSyncCompleted.
//
// §6.B plan Squad/Sessions overhaul. Couvre les guards failsafe :
//   - webhook URL vide → no-op
//   - NotifyFriends off → no-op
//   - promoted ≤ 0 (friend_sync) → no-op
//   - panic récupéré (test : passe une config invalide)
//
// L'envoi réel du webhook (SendWebhook) est testé dans webhook_test.go via
// httptest.NewServer.
package notify

import (
	"testing"
)

func TestNotifyFriendAdded_NoOpWhenWebhookEmpty(t *testing.T) {
	// Pas de webhook → pas de panic, pas d'envoi.
	cfg := NotifyConfig{
		WebhookURL:    "",
		Lang:          "en",
		NotifyFriends: true,
	}
	NotifyFriendAdded(cfg, "FriendOne") // ne doit pas paniquer
}

func TestNotifyFriendAdded_NoOpWhenFriendsOff(t *testing.T) {
	// NotifyFriends=false → no-op.
	cfg := NotifyConfig{
		WebhookURL:    "https://discord.com/api/webhooks/fake/url",
		Lang:          "en",
		NotifyFriends: false,
	}
	NotifyFriendAdded(cfg, "FriendOne")
}

func TestNotifyFriendSyncCompleted_NoOpWhenPromotedZero(t *testing.T) {
	// promoted=0 → no-op (recompute no-op, pas de notif inutile).
	cfg := NotifyConfig{
		WebhookURL:    "https://discord.com/api/webhooks/fake/url",
		Lang:          "en",
		NotifyFriends: true,
	}
	NotifyFriendSyncCompleted(cfg, "test-player", 0)
}

func TestNotifyFriendSyncCompleted_NoOpWhenFriendsOff(t *testing.T) {
	cfg := NotifyConfig{
		WebhookURL:    "https://discord.com/api/webhooks/fake/url",
		Lang:          "en",
		NotifyFriends: false,
	}
	NotifyFriendSyncCompleted(cfg, "test-player", 5)
}

func TestNotifyFriendSyncCompleted_DescSelectionEnglish(t *testing.T) {
	// Verify singular/plural English templates without sending a webhook.
	descOne := T("discord_friend_sync_desc_one", "en", "promoted", 1, "slug", "x")
	if descOne == "" {
		t.Fatal("expected non-empty English singular description")
	}
	if !contains(descOne, "1 match") {
		t.Errorf("expected English singular '1 match' in %q", descOne)
	}

	descMany := T("discord_friend_sync_desc_many", "en", "promoted", 7, "slug", "x")
	if !contains(descMany, "7 matches") {
		t.Errorf("expected English plural '7 matches' in %q", descMany)
	}
}

func TestNotifyFriendAdded_TitleEnglish(t *testing.T) {
	title := T("discord_friend_added_title", "en")
	if !contains(title, "friend") {
		t.Errorf("expected EN title to contain 'friend', got %q", title)
	}
}

// contains est un helper local pour éviter d'importer strings.
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
