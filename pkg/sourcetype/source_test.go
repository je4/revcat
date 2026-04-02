package sourcetype

import (
	"reflect"
	"testing"
	"time"

	"go.ub.unibas.ch/metastring/pkg/multilangString"
)

func TestSourceData_Interface(t *testing.T) {
	var _ Source = &SourceData{}
}

func TestSourceData_GettersSetters(t *testing.T) {
	s := &SourceData{}

	now := time.Now()
	title := &multilangString.MultiLangString{}
	abstract := &multilangString.MultiLangString{}
	persons := []Person{{Name: "John Doe"}}
	acl := map[string][]string{"read": {"user1"}}
	catalog := []string{"cat1"}
	category := []string{"categ1"}
	tags := []string{"tag1"}
	media := map[string]MediaList{"images": {{Name: "image1"}}}
	poster := &Media{Name: "poster1"}
	notes := []Note{{Title: "note1"}}
	references := []Reference{{Title: "ref1"}}
	meta := &Metalist{}
	extra := &Metalist{}
	vars := &Varlist{}
	queries := []Query{{Label: "q1"}}
	mediatype := []string{"video"}

	if err := s.SetID("123"); err != nil {
		t.Errorf("SetID() error = %v", err)
	}
	if err := s.SetSignature("sig"); err != nil {
		t.Errorf("SetSignature() error = %v", err)
	}
	if err := s.SetSignatureOriginal("sigOrig"); err != nil {
		t.Errorf("SetSignatureOriginal() error = %v", err)
	}
	if err := s.SetSource("src"); err != nil {
		t.Errorf("SetSource() error = %v", err)
	}
	if err := s.SetTitle(title); err != nil {
		t.Errorf("SetTitle() error = %v", err)
	}
	if err := s.SetSeries("series"); err != nil {
		t.Errorf("SetSeries() error = %v", err)
	}
	if err := s.SetPlace("place"); err != nil {
		t.Errorf("SetPlace() error = %v", err)
	}
	if err := s.SetDate("2023"); err != nil {
		t.Errorf("SetDate() error = %v", err)
	}
	if err := s.SetCollectionTitle("coll"); err != nil {
		t.Errorf("SetCollectionTitle() error = %v", err)
	}
	if err := s.SetPersons(persons); err != nil {
		t.Errorf("SetPersons() error = %v", err)
	}
	if err := s.SetACL(acl); err != nil {
		t.Errorf("SetACL() error = %v", err)
	}
	if err := s.SetCatalog(catalog); err != nil {
		t.Errorf("SetCatalog() error = %v", err)
	}
	if err := s.SetCategory(category); err != nil {
		t.Errorf("SetCategory() error = %v", err)
	}
	if err := s.SetTags(tags); err != nil {
		t.Errorf("SetTags() error = %v", err)
	}
	if err := s.SetMedia(media); err != nil {
		t.Errorf("SetMedia() error = %v", err)
	}
	if err := s.SetPoster(poster); err != nil {
		t.Errorf("SetPoster() error = %v", err)
	}
	if err := s.SetNotes(notes); err != nil {
		t.Errorf("SetNotes() error = %v", err)
	}
	if err := s.SetUrl("http://example.com"); err != nil {
		t.Errorf("SetUrl() error = %v", err)
	}
	if err := s.SetAbstract(abstract); err != nil {
		t.Errorf("SetAbstract() error = %v", err)
	}
	if err := s.SetReferences(references); err != nil {
		t.Errorf("SetReferences() error = %v", err)
	}
	if err := s.SetMeta(meta); err != nil {
		t.Errorf("SetMeta() error = %v", err)
	}
	if err := s.SetExtra(extra); err != nil {
		t.Errorf("SetExtra() error = %v", err)
	}
	if err := s.SetVars(vars); err != nil {
		t.Errorf("SetVars() error = %v", err)
	}
	if err := s.SetType("type"); err != nil {
		t.Errorf("SetType() error = %v", err)
	}
	if err := s.SetQueries(queries); err != nil {
		t.Errorf("SetQueries() error = %v", err)
	}
	if err := s.SetContentStr("content"); err != nil {
		t.Errorf("SetContentStr() error = %v", err)
	}
	if err := s.SetContentMime("text/plain"); err != nil {
		t.Errorf("SetContentMime() error = %v", err)
	}
	if err := s.SetHasMedia(true); err != nil {
		t.Errorf("SetHasMedia() error = %v", err)
	}
	if err := s.SetMediatype(mediatype); err != nil {
		t.Errorf("SetMediatype() error = %v", err)
	}
	if err := s.SetDateAdded(now); err != nil {
		t.Errorf("SetDateAdded() error = %v", err)
	}
	if err := s.SetTimestamp(now); err != nil {
		t.Errorf("SetTimestamp() error = %v", err)
	}
	if err := s.SetPublisher("publisher"); err != nil {
		t.Errorf("SetPublisher() error = %v", err)
	}
	if err := s.SetRights("rights"); err != nil {
		t.Errorf("SetRights() error = %v", err)
	}
	if err := s.SetLicense("license"); err != nil {
		t.Errorf("SetLicense() error = %v", err)
	}

	if s.GetID() != "123" {
		t.Errorf("GetID() = %v, want %v", s.GetID(), "123")
	}
	if s.GetSignature() != "sig" {
		t.Errorf("GetSignature() = %v, want %v", s.GetSignature(), "sig")
	}
	if s.GetSignatureOriginal() != "sigOrig" {
		t.Errorf("GetSignatureOriginal() = %v, want %v", s.GetSignatureOriginal(), "sigOrig")
	}
	if s.GetSource() != "src" {
		t.Errorf("GetSource() = %v, want %v", s.GetSource(), "src")
	}
	if s.GetTitle() != title {
		t.Errorf("GetTitle() = %v, want %v", s.GetTitle(), title)
	}
	if s.GetSeries() != "series" {
		t.Errorf("GetSeries() = %v, want %v", s.GetSeries(), "series")
	}
	if s.GetPlace() != "place" {
		t.Errorf("GetPlace() = %v, want %v", s.GetPlace(), "place")
	}
	if s.GetDate() != "2023" {
		t.Errorf("GetDate() = %v, want %v", s.GetDate(), "2023")
	}
	if s.GetCollectionTitle() != "coll" {
		t.Errorf("GetCollectionTitle() = %v, want %v", s.GetCollectionTitle(), "coll")
	}
	if !reflect.DeepEqual(s.GetPersons(), persons) {
		t.Errorf("GetPersons() = %v, want %v", s.GetPersons(), persons)
	}
	if !reflect.DeepEqual(s.GetACL(), acl) {
		t.Errorf("GetACL() = %v, want %v", s.GetACL(), acl)
	}
	if !reflect.DeepEqual(s.GetCatalog(), catalog) {
		t.Errorf("GetCatalog() = %v, want %v", s.GetCatalog(), catalog)
	}
	if !reflect.DeepEqual(s.GetCategory(), category) {
		t.Errorf("GetCategory() = %v, want %v", s.GetCategory(), category)
	}
	if !reflect.DeepEqual(s.GetTags(), tags) {
		t.Errorf("GetTags() = %v, want %v", s.GetTags(), tags)
	}
	if !reflect.DeepEqual(s.GetMedia(), media) {
		t.Errorf("GetMedia() = %v, want %v", s.GetMedia(), media)
	}
	if s.GetPoster() != poster {
		t.Errorf("GetPoster() = %v, want %v", s.GetPoster(), poster)
	}
	if !reflect.DeepEqual(s.GetNotes(), notes) {
		t.Errorf("GetNotes() = %v, want %v", s.GetNotes(), notes)
	}
	if s.GetUrl() != "http://example.com" {
		t.Errorf("GetUrl() = %v, want %v", s.GetUrl(), "http://example.com")
	}
	if s.GetAbstract() != abstract {
		t.Errorf("GetAbstract() = %v, want %v", s.GetAbstract(), abstract)
	}
	if !reflect.DeepEqual(s.GetReferences(), references) {
		t.Errorf("GetReferences() = %v, want %v", s.GetReferences(), references)
	}
	if s.GetMeta() != meta {
		t.Errorf("GetMeta() = %v, want %v", s.GetMeta(), meta)
	}
	if s.GetExtra() != extra {
		t.Errorf("GetExtra() = %v, want %v", s.GetExtra(), extra)
	}
	if s.GetVars() != vars {
		t.Errorf("GetVars() = %v, want %v", s.GetVars(), vars)
	}
	if s.GetType() != "type" {
		t.Errorf("GetType() = %v, want %v", s.GetType(), "type")
	}
	if !reflect.DeepEqual(s.GetQueries(), queries) {
		t.Errorf("GetQueries() = %v, want %v", s.GetQueries(), queries)
	}
	if s.GetContentStr() != "content" {
		t.Errorf("GetContentStr() = %v, want %v", s.GetContentStr(), "content")
	}
	if s.GetContentMime() != "text/plain" {
		t.Errorf("GetContentMime() = %v, want %v", s.GetContentMime(), "text/plain")
	}
	if s.GetHasMedia() != true {
		t.Errorf("GetHasMedia() = %v, want %v", s.GetHasMedia(), true)
	}
	if !reflect.DeepEqual(s.GetMediatype(), mediatype) {
		t.Errorf("GetMediatype() = %v, want %v", s.GetMediatype(), mediatype)
	}
	if !s.GetDateAdded().Equal(now) {
		t.Errorf("GetDateAdded() = %v, want %v", s.GetDateAdded(), now)
	}
	if !s.GetTimestamp().Equal(now) {
		t.Errorf("GetTimestamp() = %v, want %v", s.GetTimestamp(), now)
	}
	if s.GetPublisher() != "publisher" {
		t.Errorf("GetPublisher() = %v, want %v", s.GetPublisher(), "publisher")
	}
	if s.GetRights() != "rights" {
		t.Errorf("GetRights() = %v, want %v", s.GetRights(), "rights")
	}
	if s.GetLicense() != "license" {
		t.Errorf("GetLicense() = %v, want %v", s.GetLicense(), "license")
	}
}

func TestSourceData_Adders(t *testing.T) {
	s := &SourceData{}

	p := Person{Name: "Alice"}
	if err := s.AddPerson(p); err != nil {
		t.Errorf("AddPerson() error = %v", err)
	}
	if len(s.Persons) != 1 || s.Persons[0].Name != "Alice" {
		t.Errorf("AddPerson failed, got %v", s.Persons)
	}

	m := Media{Name: "video.mp4"}
	if err := s.AddMedia("video", m); err != nil {
		t.Errorf("AddMedia() error = %v", err)
	}
	if len(s.Media) != 1 || len(s.Media["video"]) != 1 || s.Media["video"][0].Name != "video.mp4" {
		t.Errorf("AddMedia failed, got %v", s.Media)
	}

	if err := s.AddMeta("author", "John"); err != nil {
		t.Errorf("AddMeta() error = %v", err)
	}
	if (*s.Meta)["author"] != "John" {
		t.Errorf("AddMeta failed, got %v", s.Meta)
	}

	if err := s.AddExtra("quality", "high"); err != nil {
		t.Errorf("AddExtra() error = %v", err)
	}
	if (*s.Extra)["quality"] != "high" {
		t.Errorf("AddExtra failed, got %v", s.Extra)
	}

	if err := s.AddVar("keywords", []string{"go", "lang"}); err != nil {
		t.Errorf("AddVar() error = %v", err)
	}
	if !reflect.DeepEqual((*s.Vars)["keywords"], []string{"go", "lang"}) {
		t.Errorf("AddVar failed, got %v", s.Vars)
	}
}
