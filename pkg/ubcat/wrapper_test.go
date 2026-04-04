package ubcat

import (
	"reflect"
	"testing"

	"github.com/je4/revcat/v2/pkg/sourcetype"
	ubschema "gitlab.switch.ch/ub-unibas/rdv2/ubcat/v2/pkg/schema"
	"go.ub.unibas.ch/metastring/pkg/metaString"
	"go.ub.unibas.ch/metastring/pkg/multilangString"
)

func TestMedUBCat_GetCategory(t *testing.T) {
	tests := []struct {
		name string
		m    *medUBCat
		want []string
	}{
		{
			name: "all nil",
			m:    &medUBCat{},
			want: nil,
		},
		{
			name: "category key exists",
			m: &medUBCat{
				Mapping: &ubschema.Mapping001{
					Subject: &ubschema.Subject{
						Local: map[string][]string{
							"category": {"cat1", "cat2"},
						},
					},
				},
			},
			want: []string{"cat1", "cat2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetCategory(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("medUBCat.GetCategory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedUBCat_GetTags(t *testing.T) {
	tests := []struct {
		name string
		m    *medUBCat
		want []string
	}{
		{
			name: "all nil",
			m:    &medUBCat{},
			want: nil,
		},
		{
			name: "mapping not nil, subject nil",
			m: &medUBCat{
				Mapping: &ubschema.Mapping001{},
			},
			want: nil,
		},
		{
			name: "subject not nil, local nil",
			m: &medUBCat{
				Mapping: &ubschema.Mapping001{
					Subject: &ubschema.Subject{},
				},
			},
			want: nil,
		},
		{
			name: "local not nil, tag key missing",
			m: &medUBCat{
				Mapping: &ubschema.Mapping001{
					Subject: &ubschema.Subject{
						Local: make(map[string][]string),
					},
				},
			},
			want: nil,
		},
		{
			name: "tag key exists",
			m: &medUBCat{
				Mapping: &ubschema.Mapping001{
					Subject: &ubschema.Subject{
						Local: map[string][]string{
							"tag": {"tag1", "tag2"},
						},
					},
				},
			},
			want: []string{"tag1", "tag2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.GetTags(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("medUBCat.GetTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMedUBCat_Poster(t *testing.T) {
	m := &medUBCat{}
	poster := &sourcetype.Media{
		Name:     "poster.jpg",
		Mimetype: "image/jpeg",
		Type:     "poster",
		Uri:      "http://example.com/poster.jpg",
		Width:    800,
		Height:   600,
		Duration: 0,
	}

	if err := m.SetPoster(poster); err != nil {
		t.Fatalf("SetPoster() error = %v", err)
	}

	got := m.GetPoster()
	if !reflect.DeepEqual(got, poster) {
		t.Errorf("GetPoster() = %v, want %v", got, poster)
	}
}

func TestMedUBCat_References(t *testing.T) {
	m := &medUBCat{}
	refs := []sourcetype.Reference{
		{
			Type:      "see",
			Title:     "Other Work",
			Signature: "SIGN-123",
		},
	}

	if err := m.SetReferences(refs); err != nil {
		t.Fatalf("SetReferences() error = %v", err)
	}

	got := m.GetReferences()
	if !reflect.DeepEqual(got, refs) {
		t.Errorf("GetReferences() = %v, want %v", got, refs)
	}

	// Test nil/empty handling
	m = &medUBCat{}
	if got := m.GetReferences(); got != nil {
		t.Errorf("GetReferences() with nil structures = %v, want nil", got)
	}
}

func TestMedUBCat_Extra(t *testing.T) {
	m := &medUBCat{}
	extra := &sourcetype.Metalist{
		"key1": "val1",
		"key2": "val2",
	}

	if err := m.SetExtra(extra); err != nil {
		t.Fatalf("SetExtra() error = %v", err)
	}

	if m.RawJson["extra"] == nil {
		t.Fatal("RawJson[\"extra\"] should not be nil")
	}

	got := m.GetExtra()
	if !reflect.DeepEqual(got, extra) {
		t.Errorf("GetExtra() = %v, want %v", got, extra)
	}

	if err := m.AddExtra("key3", "val3"); err != nil {
		t.Fatalf("AddExtra() error = %v", err)
	}

	got = m.GetExtra()
	if (*got)["key3"] != "val3" {
		t.Errorf("AddExtra failed, key3 not found in %v", got)
	}
}

func TestMedUBCat_Meta(t *testing.T) {
	m := &medUBCat{}
	meta := &sourcetype.Metalist{
		"m1": "v1",
		"m2": "v2",
	}

	if err := m.SetMeta(meta); err != nil {
		t.Fatalf("SetMeta() error = %v", err)
	}

	if m.RawJson["meta"] == nil {
		t.Fatal("RawJson[\"meta\"] should not be nil")
	}

	got := m.GetMeta()
	if !reflect.DeepEqual(got, meta) {
		t.Errorf("GetMeta() = %v, want %v", got, meta)
	}

	if err := m.AddMeta("m3", "v3"); err != nil {
		t.Fatalf("AddMeta() error = %v", err)
	}

	got = m.GetMeta()
	if (*got)["m3"] != "v3" {
		t.Errorf("AddMeta failed, m3 not found in %v", got)
	}
}

func TestMedUBCat_Vars(t *testing.T) {
	m := &medUBCat{}
	vars := &sourcetype.Varlist{
		"key1": {"val1"},
		"key2": {"val2a", "val2b"},
	}

	if err := m.SetVars(vars); err != nil {
		t.Fatalf("SetVars() error = %v", err)
	}

	got := m.GetVars()
	if !reflect.DeepEqual(got, vars) {
		t.Errorf("GetVars() = %v, want %v", got, vars)
	}

	if err := m.AddVar("key3", []string{"val3"}); err != nil {
		t.Fatalf("AddVar() error = %v", err)
	}

	got = m.GetVars()
	if !reflect.DeepEqual((*got)["key3"], []string{"val3"}) {
		t.Errorf("AddVar failed, key3 not found in %v", got)
	}

	// Test nil/empty
	if err := m.SetVars(nil); err != nil {
		t.Fatalf("SetVars(nil) error = %v", err)
	}
	got = m.GetVars()
	if len(*got) != 0 {
		t.Errorf("GetVars() after SetVars(nil) = %v, want empty", got)
	}
}

func TestMedUBCat_Abstract(t *testing.T) {
	m := &medUBCat{}
	abstractStr := "Dies ist ein Abstract"
	abstract := multilangString.NewMultiLangString(metaString.NewMetaString(abstractStr))

	if err := m.SetAbstract(abstract); err != nil {
		t.Fatalf("SetAbstract() error = %v", err)
	}

	got := m.GetAbstract()
	if got.String().String() != abstractStr {
		t.Errorf("GetAbstract() = %v, want %v", got.String().String(), abstractStr)
	}

	// Test nil handling
	if err := m.SetAbstract(nil); err == nil {
		t.Error("SetAbstract(nil) should return error")
	}
}

func TestSourceDataAndUBSchema001(t *testing.T) {
	// 1. Initialisiere medUBCat (basiert auf UBSchema001)
	m := &medUBCat{}

	// 2. Erstelle Testdaten
	id := "test-id-123"
	signature := "SIGN-001"
	titleStr := "Test Titel"
	date := "2024"
	abstractStr := "Dies ist ein Abstract"
	abstract := multilangString.NewMultiLangString(metaString.NewMetaString(abstractStr))

	persons := []sourcetype.Person{
		{
			Name: "Mustermann, Erika",
			Role: "author",
		},
	}

	// 3. Setze Daten in medUBCat
	if err := m.SetID(id); err != nil {
		t.Fatalf("SetID() error = %v", err)
	}
	if err := m.SetSignature(signature); err != nil {
		t.Fatalf("SetSignature() error = %v", err)
	}

	title := multilangString.NewMultiLangString(metaString.NewMetaString(titleStr))
	if err := m.SetTitle(title.String()); err != nil {
		t.Fatalf("SetTitle() error = %v", err)
	}

	if err := m.SetDate(date); err != nil {
		t.Fatalf("SetDate() error = %v", err)
	}

	if err := m.SetAbstract(abstract); err != nil {
		t.Fatalf("SetAbstract() error = %v", err)
	}

	if err := m.SetPersons(persons); err != nil {
		t.Fatalf("SetPersons() error = %v", err)
	}

	// 4. Verifiziere Get-Methoden von medUBCat
	if m.GetID() != id {
		t.Errorf("GetID() = %v, want %v", m.GetID(), id)
	}
	if m.GetSignature() != signature {
		t.Errorf("GetSignature() = %v, want %v", m.GetSignature(), signature)
	}
	if m.GetTitle().String() != titleStr {
		t.Errorf("GetTitle() = %v, want %v", m.GetTitle().String(), titleStr)
	}
	if m.GetDate() != date {
		t.Errorf("GetDate() = %v, want %v", m.GetDate(), date)
	}
	if m.GetAbstract().String().String() != abstractStr {
		t.Errorf("GetAbstract() = %v, want %v", m.GetAbstract().String().String(), abstractStr)
	}
	if !reflect.DeepEqual(m.GetPersons(), persons) {
		t.Errorf("GetPersons() = %v, want %v", m.GetPersons(), persons)
	}

	// 5. Vergleich mit SourceData (andere Implementierung des Source-Interfaces)
	sd := &sourcetype.SourceData{}
	sd.SetID(id)
	sd.SetSignature(signature)
	sd.SetTitle(title.String())
	sd.SetDate(date)
	sd.SetAbstract(abstract)
	sd.SetPersons(persons)

	// Prüfe ob beide Implementierungen die gleichen Werte über das Interface liefern
	var s1 sourcetype.Source = m
	var s2 sourcetype.Source = sd

	if s1.GetID() != s2.GetID() {
		t.Errorf("m.GetID() [%v] != sd.GetID() [%v]", s1.GetID(), s2.GetID())
	}
	if s1.GetSignature() != s2.GetSignature() {
		t.Errorf("m.GetSignature() [%v] != sd.GetSignature() [%v]", s1.GetSignature(), s2.GetSignature())
	}
	if s1.GetTitle().String() != s2.GetTitle().String() {
		t.Errorf("m.GetTitle() [%v] != sd.GetTitle() [%v]", s1.GetTitle(), s2.GetTitle())
	}
	if s1.GetDate() != s2.GetDate() {
		t.Errorf("m.GetDate() [%v] != sd.GetDate() [%v]", s1.GetDate(), s2.GetDate())
	}
	if s1.GetAbstract().String().String() != s2.GetAbstract().String().String() {
		t.Errorf("m.GetAbstract() [%v] != sd.GetAbstract() [%v]", s1.GetAbstract(), s2.GetAbstract())
	}
	if !reflect.DeepEqual(s1.GetPersons(), s2.GetPersons()) {
		t.Errorf("m.GetPersons() [%v] != sd.GetPersons() [%v]", s1.GetPersons(), s2.GetPersons())
	}
}
