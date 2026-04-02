package sourcetype

import (
	"time"

	"go.ub.unibas.ch/metastring/pkg/multilangString"
)

func (s *SourceData) SetID(id string) error {
	s.ID = id
	return nil
}

func (s *SourceData) SetSignature(signature string) error {
	s.Signature = signature
	return nil
}

func (s *SourceData) SetSignatureOriginal(signatureOriginal string) error {
	s.SignatureOriginal = signatureOriginal
	return nil
}

func (s *SourceData) SetSource(source string) error {
	s.Source = source
	return nil
}

func (s *SourceData) SetTitle(title *multilangString.MultiLangString) error {
	s.Title = title
	return nil
}

func (s *SourceData) SetSeries(series string) error {
	s.Series = series
	return nil
}

func (s *SourceData) SetPlace(place string) error {
	s.Place = place
	return nil
}

func (s *SourceData) SetDate(date string) error {
	s.Date = date
	return nil
}

func (s *SourceData) SetCollectionTitle(collectionTitle string) error {
	s.CollectionTitle = collectionTitle
	return nil
}

func (s *SourceData) SetPersons(persons []Person) error {
	s.Persons = persons
	return nil
}

func (s *SourceData) SetACL(acl map[string][]string) error {
	s.ACL = acl
	return nil
}

func (s *SourceData) SetCatalog(catalog []string) error {
	s.Catalog = catalog
	return nil
}

func (s *SourceData) SetCategory(category []string) error {
	s.Category = category
	return nil
}

func (s *SourceData) SetTags(tags []string) error {
	s.Tags = tags
	return nil
}

func (s *SourceData) SetMedia(media map[string]MediaList) error {
	s.Media = media
	return nil
}

func (s *SourceData) SetPoster(poster *Media) error {
	s.Poster = poster
	return nil
}

func (s *SourceData) SetNotes(notes []Note) error {
	s.Notes = notes
	return nil
}

func (s *SourceData) SetUrl(url string) error {
	s.Url = url
	return nil
}

func (s *SourceData) SetAbstract(abstract *multilangString.MultiLangString) error {
	s.Abstract = abstract
	return nil
}

func (s *SourceData) SetReferences(references []Reference) error {
	s.References = references
	return nil
}

func (s *SourceData) SetMeta(meta *Metalist) error {
	s.Meta = meta
	return nil
}

func (s *SourceData) SetExtra(extra *Metalist) error {
	s.Extra = extra
	return nil
}

func (s *SourceData) SetVars(vars *Varlist) error {
	s.Vars = vars
	return nil
}

func (s *SourceData) SetType(t string) error {
	s.Type = t
	return nil
}

func (s *SourceData) SetQueries(queries []Query) error {
	s.Queries = queries
	return nil
}

func (s *SourceData) SetContentStr(contentStr string) error {
	s.ContentStr = contentStr
	return nil
}

func (s *SourceData) SetContentMime(contentMime string) error {
	s.ContentMime = contentMime
	return nil
}

func (s *SourceData) SetHasMedia(hasMedia bool) error {
	s.HasMedia = hasMedia
	return nil
}

func (s *SourceData) SetMediatype(mediatype []string) error {
	s.Mediatype = mediatype
	return nil
}

func (s *SourceData) SetDateAdded(dateAdded time.Time) error {
	s.DateAdded = dateAdded
	return nil
}

func (s *SourceData) SetTimestamp(timestamp time.Time) error {
	s.Timestamp = timestamp
	return nil
}

func (s *SourceData) SetPublisher(publisher string) error {
	s.Publisher = publisher
	return nil
}

func (s *SourceData) SetRights(rights string) error {
	s.Rights = rights
	return nil
}

func (s *SourceData) SetLicense(license string) error {
	s.License = license
	return nil
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
