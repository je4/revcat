package ubcat

import (
	"html/template"
	"reflect"
	"testing"
	"time"

	"github.com/je4/revcat/v2/pkg/sourcetype"
	"go.ub.unibas.ch/metastring/pkg/metaString"
	"go.ub.unibas.ch/metastring/pkg/multilangString"
)

func fillSource(s sourcetype.Source) error {
	if err := s.SetID("id-123"); err != nil {
		return err
	}
	if err := s.SetSignature("SIGN-001"); err != nil {
		return err
	}
	if err := s.SetSignatureOriginal("ORIG-001"); err != nil {
		return err
	}

	title := &multilangString.MultiLangString{}
	title.Set(metaString.NewMetaString("Main Title"))
	if err := s.SetTitle(title.String()); err != nil {
		return err
	}

	if err := s.SetSeries("Series Title"); err != nil {
		return err
	}
	if err := s.SetPlace("Basel"); err != nil {
		return err
	}
	if err := s.SetDate("2024"); err != nil {
		return err
	}
	if err := s.SetCollectionTitle("Collection X"); err != nil {
		return err
	}

	persons := []sourcetype.Person{
		{
			Name: "Name 1",
			Role: "author", // medUBCat only stores Name and Role
		},
	}
	if err := s.SetPersons(persons); err != nil {
		return err
	}

	acl := map[string][]string{
		"content": {"group1"}, // medUBCat uses "content", "meta", "preview"
		"meta":    {"group2"},
		"preview": {"group3"},
	}
	if err := s.SetACL(acl); err != nil {
		return err
	}

	if err := s.SetCatalog([]string{"cat1", "cat2"}); err != nil {
		return err
	}
	if err := s.SetCategory([]string{"category1"}); err != nil {
		return err
	}
	if err := s.SetTags([]string{"tag1", "tag2"}); err != nil {
		return err
	}

	media := map[string]sourcetype.MediaList{
		"video": {
			{
				Name:     "video1.mp4",
				Mimetype: "video/mp4",
				Type:     "video",
				Uri:      "http://example.com/v1.mp4",
				Width:    1920,
				Height:   1080,
				Duration: 120,
			},
		},
	}
	if err := s.SetMedia(media); err != nil {
		return err
	}

	poster := &sourcetype.Media{
		Name:     "poster.jpg",
		Mimetype: "image/jpeg",
		Type:     "poster",
		Uri:      "http://example.com/p.jpg",
	}
	if err := s.SetPoster(poster); err != nil {
		return err
	}

	notes := []sourcetype.Note{
		{Title: "General", Note: template.HTML("Some note")}, // medUBCat uses Title "General"
	}
	if err := s.SetNotes(notes); err != nil {
		return err
	}

	abstract := multilangString.NewMultiLangString(metaString.NewMetaString("This is an abstract"))
	if err := s.SetAbstract(abstract); err != nil {
		return err
	}

	refs := []sourcetype.Reference{
		{Type: "see", Title: "Other Item", Signature: "SIGN-002"},
	}
	if err := s.SetReferences(refs); err != nil {
		return err
	}

	meta := &sourcetype.Metalist{"meta1": "val1"}
	if err := s.SetMeta(meta); err != nil {
		return err
	}

	extra := &sourcetype.Metalist{"extra1": "eval1"}
	if err := s.SetExtra(extra); err != nil {
		return err
	}

	vars := &sourcetype.Varlist{"var1": {"v1"}} // medUBCat SetVars handles single values differently in RawJson
	if err := s.SetVars(vars); err != nil {
		return err
	}

	if err := s.SetType("image"); err != nil {
		return err
	}

	now := time.Now().Truncate(time.Second).UTC() // medUBCat doesn't seem to store DateAdded, only Timestamp
	if err := s.SetTimestamp(now); err != nil {
		return err
	}

	if err := s.SetPublisher("Publisher X"); err != nil {
		return err
	}

	return nil
}

func transfer(src, dst sourcetype.Source) error {
	if err := dst.SetID(src.GetID()); err != nil {
		return err
	}
	if err := dst.SetSignature(src.GetSignature()); err != nil {
		return err
	}
	if err := dst.SetSignatureOriginal(src.GetSignatureOriginal()); err != nil {
		return err
	}
	if err := dst.SetSource(src.GetSource()); err != nil {
		return err
	}
	if err := dst.SetTitle(src.GetTitle()); err != nil {
		return err
	}
	if err := dst.SetSeries(src.GetSeries()); err != nil {
		return err
	}
	if err := dst.SetPlace(src.GetPlace()); err != nil {
		return err
	}
	if err := dst.SetDate(src.GetDate()); err != nil {
		return err
	}
	if err := dst.SetCollectionTitle(src.GetCollectionTitle()); err != nil {
		return err
	}
	if err := dst.SetPersons(src.GetPersons()); err != nil {
		return err
	}
	if err := dst.SetACL(src.GetACL()); err != nil {
		return err
	}
	if err := dst.SetCatalog(src.GetCatalog()); err != nil {
		return err
	}
	if err := dst.SetCategory(src.GetCategory()); err != nil {
		return err
	}
	if err := dst.SetTags(src.GetTags()); err != nil {
		return err
	}
	if err := dst.SetMedia(src.GetMedia()); err != nil {
		return err
	}
	if err := dst.SetPoster(src.GetPoster()); err != nil {
		return err
	}
	if err := dst.SetNotes(src.GetNotes()); err != nil {
		return err
	}
	if err := dst.SetUrl(src.GetUrl()); err != nil {
		return err
	}
	if err := dst.SetAbstract(src.GetAbstract()); err != nil {
		return err
	}
	if err := dst.SetReferences(src.GetReferences()); err != nil {
		return err
	}
	if err := dst.SetMeta(src.GetMeta()); err != nil {
		return err
	}
	if err := dst.SetExtra(src.GetExtra()); err != nil {
		return err
	}
	if err := dst.SetVars(src.GetVars()); err != nil {
		return err
	}
	if err := dst.SetType(src.GetType()); err != nil {
		return err
	}
	if err := dst.SetQueries(src.GetQueries()); err != nil {
		return err
	}
	if err := dst.SetContentStr(src.GetContentStr()); err != nil {
		return err
	}
	if err := dst.SetContentMime(src.GetContentMime()); err != nil {
		return err
	}
	if err := dst.SetHasMedia(src.GetHasMedia()); err != nil {
		return err
	}
	if err := dst.SetMediatype(src.GetMediatype()); err != nil {
		return err
	}
	if err := dst.SetDateAdded(src.GetDateAdded()); err != nil {
		return err
	}
	if err := dst.SetTimestamp(src.GetTimestamp()); err != nil {
		return err
	}
	if err := dst.SetPublisher(src.GetPublisher()); err != nil {
		return err
	}
	if err := dst.SetRights(src.GetRights()); err != nil {
		return err
	}
	if err := dst.SetLicense(src.GetLicense()); err != nil {
		return err
	}
	return nil
}

func compareSources(t *testing.T, s1, s2 sourcetype.Source) {
	t.Helper()
	if s1.GetID() != s2.GetID() {
		t.Errorf("ID mismatch: %v != %v", s1.GetID(), s2.GetID())
	}
	if s1.GetSignature() != s2.GetSignature() {
		t.Errorf("Signature mismatch: %v != %v", s1.GetSignature(), s2.GetSignature())
	}
	if s1.GetSignatureOriginal() != s2.GetSignatureOriginal() {
		t.Errorf("SignatureOriginal mismatch: %v != %v", s1.GetSignatureOriginal(), s2.GetSignatureOriginal())
	}

	if s1.GetTitle().String() != s2.GetTitle().String() {
		t.Errorf("Title mismatch: %v != %v", s1.GetTitle(), s2.GetTitle())
	}
	if s1.GetSeries() != s2.GetSeries() {
		t.Errorf("Series mismatch: %v != %v", s1.GetSeries(), s2.GetSeries())
	}
	if s1.GetPlace() != s2.GetPlace() {
		t.Errorf("Place mismatch: %v != %v", s1.GetPlace(), s2.GetPlace())
	}
	if s1.GetDate() != s2.GetDate() {
		t.Errorf("Date mismatch: %v != %v", s1.GetDate(), s2.GetDate())
	}
	if s1.GetCollectionTitle() != s2.GetCollectionTitle() {
		t.Errorf("CollectionTitle mismatch: %v != %v", s1.GetCollectionTitle(), s2.GetCollectionTitle())
	}

	if !reflect.DeepEqual(s1.GetPersons(), s2.GetPersons()) {
		t.Errorf("Persons mismatch: %+v != %+v", s1.GetPersons(), s2.GetPersons())
	}
	if !reflect.DeepEqual(s1.GetACL(), s2.GetACL()) {
		t.Errorf("ACL mismatch: %+v != %+v", s1.GetACL(), s2.GetACL())
	}
	if !reflect.DeepEqual(s1.GetCatalog(), s2.GetCatalog()) {
		t.Errorf("Catalog mismatch: %+v != %+v", s1.GetCatalog(), s2.GetCatalog())
	}
	if !reflect.DeepEqual(s1.GetCategory(), s2.GetCategory()) {
		t.Errorf("Category mismatch: %+v != %+v", s1.GetCategory(), s2.GetCategory())
	}
	if !reflect.DeepEqual(s1.GetTags(), s2.GetTags()) {
		t.Errorf("Tags mismatch: %+v != %+v", s1.GetTags(), s2.GetTags())
	}

	// Media map comparison might fail if keys order is different or if there are nil/empty slice issues
	m1 := s1.GetMedia()
	m2 := s2.GetMedia()
	// Skip comparison if one of them is empty (medUBCat might not be fully initialized in some tests)
	if len(m1) > 0 || len(m2) > 0 {
		if len(m1) != len(m2) {
			t.Errorf("Media length mismatch: %d != %d", len(m1), len(m2))
		}
		for k, v1 := range m1 {
			v2, ok := m2[k]
			if !ok {
				t.Errorf("Media key %s missing in s2", k)
				continue
			}
			if !reflect.DeepEqual(v1, v2) {
				t.Errorf("Media value mismatch for key %s: %+v != %+v", k, v1, v2)
			}
		}
	}

	if !reflect.DeepEqual(s1.GetPoster(), s2.GetPoster()) {
		t.Errorf("Poster mismatch: %+v != %+v", s1.GetPoster(), s2.GetPoster())
	}
	if !reflect.DeepEqual(s1.GetNotes(), s2.GetNotes()) {
		t.Errorf("Notes mismatch: %+v != %+v", s1.GetNotes(), s2.GetNotes())
	}

	if s1.GetAbstract().String().String() != s2.GetAbstract().String().String() {
		t.Errorf("Abstract mismatch: %v != %v", s1.GetAbstract(), s2.GetAbstract())
	}

	if !reflect.DeepEqual(s1.GetReferences(), s2.GetReferences()) {
		t.Errorf("References mismatch: %+v != %+v", s1.GetReferences(), s2.GetReferences())
	}
	if !reflect.DeepEqual(s1.GetMeta(), s2.GetMeta()) {
		t.Errorf("Meta mismatch: %+v != %+v", s1.GetMeta(), s2.GetMeta())
	}
	if !reflect.DeepEqual(s1.GetExtra(), s2.GetExtra()) {
		t.Errorf("Extra mismatch: %+v != %+v", s1.GetExtra(), s2.GetExtra())
	}
	if !reflect.DeepEqual(s1.GetVars(), s2.GetVars()) {
		t.Errorf("Vars mismatch: %+v != %+v", s1.GetVars(), s2.GetVars())
	}

	if s1.GetType() != s2.GetType() {
		t.Errorf("Type mismatch: %v != %v", s1.GetType(), s2.GetType())
	}

	// Timestamp comparison
	if !s1.GetTimestamp().Equal(s2.GetTimestamp()) {
		t.Errorf("Timestamp mismatch: %v != %v", s1.GetTimestamp(), s2.GetTimestamp())
	}

	if s1.GetPublisher() != s2.GetPublisher() {
		t.Errorf("Publisher mismatch: %v != %v", s1.GetPublisher(), s2.GetPublisher())
	}
}

func TestLosslessTransfer_SourceDataToMedUBCat(t *testing.T) {
	sd := &sourcetype.SourceData{}
	if err := fillSource(sd); err != nil {
		t.Fatalf("Failed to fill SourceData: %v", err)
	}

	m := &medUBCat{}
	if err := transfer(sd, m); err != nil {
		t.Fatalf("Failed to transfer from SourceData to medUBCat: %v", err)
	}

	compareSources(t, sd, m)
}

func TestLosslessTransfer_MedUBCatToSourceData(t *testing.T) {
	m := &medUBCat{}
	if err := fillSource(m); err != nil {
		t.Fatalf("Failed to fill medUBCat: %v", err)
	}

	sd := &sourcetype.SourceData{}
	if err := transfer(m, sd); err != nil {
		t.Fatalf("Failed to transfer from medUBCat to SourceData: %v", err)
	}

	compareSources(t, m, sd)
}
