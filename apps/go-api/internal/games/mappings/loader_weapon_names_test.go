package mappings

import "testing"

func TestLoadWeaponNames_Valid(t *testing.T) {
	raw := []byte(`
[meta]
title_slug     = "halo_5"
schema_version = 2

[weapons]
h5_frag_grenade = { en = "Frag Grenade" }
h5_light_rifle  = { en = "Light Rifle" }
h5_railgun      = { en = "Railgun" }
`)
	set, err := LoadWeaponNamesFromBytes("weapon_names.toml", raw)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if set.TitleSlug() != "halo_5" {
		t.Errorf("title_slug: got %q", set.TitleSlug())
	}
	if set.SchemaVersion() != 2 {
		t.Errorf("schema_version: got %d", set.SchemaVersion())
	}
	names := set.Names()
	if len(names) != 3 {
		t.Fatalf("names count: got %d, want 3", len(names))
	}
	if got := names["h5_frag_grenade"]; got.En != "Frag Grenade" || got.Fr != "Frag Grenade" {
		t.Errorf("h5_frag_grenade: got %+v", got)
	}
	// Legacy fields mirror the English label for compatibility.
	if got := names["h5_railgun"]; got.En != "Railgun" || got.Fr != "Railgun" {
		t.Errorf("h5_railgun: got %+v", got)
	}
}

func TestWeaponNameSet_NilIsSafe(t *testing.T) {
	var set *WeaponNameSet
	if n := len(set.Names()); n != 0 {
		t.Errorf("nil Names must be empty: got %d", n)
	}
	if set.TitleSlug() != "" || set.SchemaVersion() != 0 {
		t.Error("nil accessors must return zero values")
	}
}

func TestLoadWeaponNames_Invalid(t *testing.T) {
	cases := map[string]string{
		"meta manquant": `
[weapons]
k = { en = "A" }
`,
		"schema_version zero": `
[meta]
title_slug = "halo_5"
[weapons]
k = { en = "A" }
`,
		"en vide": `
[meta]
title_slug     = "halo_5"
schema_version = 1
[weapons]
k = { en = "" }
`,
		"aucune arme": `
[meta]
title_slug     = "halo_5"
schema_version = 1
`,
	}
	for name, raw := range cases {
		if _, err := LoadWeaponNamesFromBytes("t.toml", []byte(raw)); err == nil {
			t.Errorf("%s: attendu une erreur", name)
		}
	}
}
