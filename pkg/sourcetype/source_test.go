package sourcetype

import (
	"reflect"
	"testing"
	"time"

	"go.ub.unibas.ch/metastring/pkg/multilangString"
)

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

	s.SetID("123")
	s.SetSignature("sig")
	s.SetSignatureOriginal("sigOrig")
	s.SetSource("src")
	s.SetTitle(title)
	s.SetSeries("series")
	s.SetPlace("place")
	s.SetDate("2023")
	s.SetCollectionTitle("coll")
	s.SetPersons(persons)
	s.SetACL(acl)
	s.SetCatalog(catalog)
	s.SetCategory(category)
	s.SetTags(tags)
	s.SetMedia(media)
	s.SetPoster(poster)
	s.SetNotes(notes)
	s.SetUrl("http://example.com")
	s.SetAbstract(abstract)
	s.SetReferences(references)
	s.SetMeta(meta)
	s.SetExtra(extra)
	s.SetVars(vars)
	s.SetType("type")
	s.SetQueries(queries)
	s.SetContentStr("content")
	s.SetContentMime("text/plain")
	s.SetHasMedia(true)
	s.SetMediatype(mediatype)
	s.SetDateAdded(now)
	s.SetTimestamp(now)
	s.SetPublisher("publisher")
	s.SetRights("rights")
	s.SetLicense("license")

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
	s.AddPerson(p)
	if len(s.Persons) != 1 || s.Persons[0].Name != "Alice" {
		t.Errorf("AddPerson failed, got %v", s.Persons)
	}

	m := Media{Name: "video.mp4"}
	s.AddMedia("video", m)
	if len(s.Media) != 1 || len(s.Media["video"]) != 1 || s.Media["video"][0].Name != "video.mp4" {
		t.Errorf("AddMedia failed, got %v", s.Media)
	}

	s.AddMeta("author", "John")
	if (*s.Meta)["author"] != "John" {
		t.Errorf("AddMeta failed, got %v", s.Meta)
	}

	s.AddExtra("quality", "high")
	if (*s.Extra)["quality"] != "high" {
		t.Errorf("AddExtra failed, got %v", s.Extra)
	}

	s.AddVar("keywords", []string{"go", "lang"})
	if !reflect.DeepEqual((*s.Vars)["keywords"], []string{"go", "lang"}) {
		t.Errorf("AddVar failed, got %v", s.Vars)
	}
}
