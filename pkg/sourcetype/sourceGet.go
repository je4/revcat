package sourcetype

import (
	"time"

	"go.ub.unibas.ch/metastring/pkg/metaString"
	"go.ub.unibas.ch/metastring/pkg/multilangString"
)

func (s *SourceData) GetID() string {
	return s.ID
}

func (s *SourceData) GetSignature() string {
	return s.Signature
}

func (s *SourceData) GetSignatureOriginal() string {
	return s.SignatureOriginal
}

func (s *SourceData) GetSource() string {
	return s.Source
}

func (s *SourceData) GetTitle() *metaString.MetaString {
	if s.Title == nil {
		return metaString.NewMetaString("")
	}
	return s.Title.String()
}

func (s *SourceData) GetSeries() string {
	return s.Series
}

func (s *SourceData) GetPlace() string {
	return s.Place
}

func (s *SourceData) GetDate() string {
	return s.Date
}

func (s *SourceData) GetCollectionTitle() string {
	return s.CollectionTitle
}

func (s *SourceData) GetPersons() []Person {
	return s.Persons
}

func (s *SourceData) GetACL() map[string][]string {
	return s.ACL
}

func (s *SourceData) GetCatalog() []string {
	return s.Catalog
}

func (s *SourceData) GetCategory() []string {
	return s.Category
}

func (s *SourceData) GetTags() []string {
	return s.Tags
}

func (s *SourceData) GetMedia() map[string]MediaList {
	return s.Media
}

func (s *SourceData) GetPoster() *Media {
	if s.Poster == nil {
		return &Media{}
	}
	return s.Poster
}

func (s *SourceData) GetNotes() []Note {
	return s.Notes
}

func (s *SourceData) GetUrl() string {
	return s.Url
}

func (s *SourceData) GetAbstract() *multilangString.MultiLangString {
	if s.Abstract == nil {
		return multilangString.NewMultiLangString()
	}
	return s.Abstract
}

func (s *SourceData) GetReferences() []Reference {
	return s.References
}

func (s *SourceData) GetMeta() *Metalist {
	if s.Meta == nil {
		return &Metalist{}
	}
	return s.Meta
}

func (s *SourceData) GetExtra() *Metalist {
	if s.Extra == nil {
		return &Metalist{}
	}
	return s.Extra
}

func (s *SourceData) GetVars() *Varlist {
	if s.Vars == nil {
		return &Varlist{}
	}
	return s.Vars
}

func (s *SourceData) GetType() string {
	return s.Type
}

func (s *SourceData) GetQueries() []Query {
	return s.Queries
}

func (s *SourceData) GetContentStr() string {
	return s.ContentStr
}

func (s *SourceData) GetContentMime() string {
	return s.ContentMime
}

func (s *SourceData) GetHasMedia() bool {
	return s.HasMedia
}

func (s *SourceData) GetMediatype() []string {
	return s.Mediatype
}

func (s *SourceData) GetDateAdded() time.Time {
	return s.DateAdded
}

func (s *SourceData) GetTimestamp() time.Time {
	return s.Timestamp
}

func (s *SourceData) GetPublisher() string {
	return s.Publisher
}

func (s *SourceData) GetRights() string {
	return s.Rights
}

func (s *SourceData) GetLicense() string {
	return s.License
}
