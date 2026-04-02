package sourcetype

import (
	"time"

	"go.ub.unibas.ch/metastring/pkg/multilangString"
)

func (s *SourceData) SetID(id string) {
	s.ID = id
}

func (s *SourceData) SetSignature(signature string) {
	s.Signature = signature
}

func (s *SourceData) SetSignatureOriginal(signatureOriginal string) {
	s.SignatureOriginal = signatureOriginal
}

func (s *SourceData) SetSource(source string) {
	s.Source = source
}

func (s *SourceData) SetTitle(title *multilangString.MultiLangString) {
	s.Title = title
}

func (s *SourceData) SetSeries(series string) {
	s.Series = series
}

func (s *SourceData) SetPlace(place string) {
	s.Place = place
}

func (s *SourceData) SetDate(date string) {
	s.Date = date
}

func (s *SourceData) SetCollectionTitle(collectionTitle string) {
	s.CollectionTitle = collectionTitle
}

func (s *SourceData) SetPersons(persons []Person) {
	s.Persons = persons
}

func (s *SourceData) SetACL(acl map[string][]string) {
	s.ACL = acl
}

func (s *SourceData) SetCatalog(catalog []string) {
	s.Catalog = catalog
}

func (s *SourceData) SetCategory(category []string) {
	s.Category = category
}

func (s *SourceData) SetTags(tags []string) {
	s.Tags = tags
}

func (s *SourceData) SetMedia(media map[string]MediaList) {
	s.Media = media
}

func (s *SourceData) SetPoster(poster *Media) {
	s.Poster = poster
}

func (s *SourceData) SetNotes(notes []Note) {
	s.Notes = notes
}

func (s *SourceData) SetUrl(url string) {
	s.Url = url
}

func (s *SourceData) SetAbstract(abstract *multilangString.MultiLangString) {
	s.Abstract = abstract
}

func (s *SourceData) SetReferences(references []Reference) {
	s.References = references
}

func (s *SourceData) SetMeta(meta *Metalist) {
	s.Meta = meta
}

func (s *SourceData) SetExtra(extra *Metalist) {
	s.Extra = extra
}

func (s *SourceData) SetVars(vars *Varlist) {
	s.Vars = vars
}

func (s *SourceData) SetType(t string) {
	s.Type = t
}

func (s *SourceData) SetQueries(queries []Query) {
	s.Queries = queries
}

func (s *SourceData) SetContentStr(contentStr string) {
	s.ContentStr = contentStr
}

func (s *SourceData) SetContentMime(contentMime string) {
	s.ContentMime = contentMime
}

func (s *SourceData) SetHasMedia(hasMedia bool) {
	s.HasMedia = hasMedia
}

func (s *SourceData) SetMediatype(mediatype []string) {
	s.Mediatype = mediatype
}

func (s *SourceData) SetDateAdded(dateAdded time.Time) {
	s.DateAdded = dateAdded
}

func (s *SourceData) SetTimestamp(timestamp time.Time) {
	s.Timestamp = timestamp
}

func (s *SourceData) SetPublisher(publisher string) {
	s.Publisher = publisher
}

func (s *SourceData) SetRights(rights string) {
	s.Rights = rights
}

func (s *SourceData) SetLicense(license string) {
	s.License = license
}

func (s *SourceData) AddPerson(p Person) {
	s.Persons = append(s.Persons, p)
}

func (s *SourceData) AddMedia(kind string, m Media) {
	if s.Media == nil {
		s.Media = make(map[string]MediaList)
	}
	if _, ok := s.Media[kind]; !ok {
		s.Media[kind] = MediaList{}
	}
	s.Media[kind] = append(s.Media[kind], m)
}

func (s *SourceData) AddMeta(key, value string) {
	if s.Meta == nil {
		s.Meta = &Metalist{}
	}
	(*s.Meta)[key] = value
}

func (s *SourceData) AddExtra(key, value string) {
	if s.Extra == nil {
		s.Extra = &Metalist{}
	}
	(*s.Extra)[key] = value
}

func (s *SourceData) AddVar(key string, value []string) {
	if s.Vars == nil {
		s.Vars = &Varlist{}
	}
	s.Vars.Append(key, value)
}
