package ubcat

import (
	"html/template"
	"time"

	"github.com/pkg/errors"

	"github.com/je4/revcat/v2/pkg/sourcetype"
	ubschema "gitlab.switch.ch/ub-unibas/rdv2/ubcat/v2/pkg/schema"
	"go.ub.unibas.ch/metastring/pkg/metaString"
	"go.ub.unibas.ch/metastring/pkg/multilangString"
)

type medUBCat ubschema.UBSchema001

func (m *medUBCat) cast() *ubschema.UBSchema001 {
	return (*ubschema.UBSchema001)(m)
}

func (m *medUBCat) GetID() string {
	return m.Id
}

func (m *medUBCat) SetID(id string) error {
	m.Id = id
	return nil
}

func (m *medUBCat) GetSignature() string {
	if m.Mapping == nil || len(m.Mapping.RecordIdentifier) == 0 {
		return ""
	}
	return m.Mapping.RecordIdentifier[0]
}

func (m *medUBCat) SetSignature(signature string) error {
	if m.Mapping == nil {
		m.Mapping = &ubschema.Mapping001{}
	}
	m.Mapping.RecordIdentifier = []string{signature}
	return nil
}

func (m *medUBCat) GetSignatureOriginal() string {
	if m.Mapping == nil || len(m.Mapping.RecordIdentifier) < 2 {
		return ""
	}
	return m.Mapping.RecordIdentifier[1]
}

func (m *medUBCat) SetSignatureOriginal(signatureOriginal string) error {
	if m.Mapping == nil {
		m.Mapping = &ubschema.Mapping001{}
	}
	if len(m.Mapping.RecordIdentifier) < 2 {
		for len(m.Mapping.RecordIdentifier) < 2 {
			m.Mapping.RecordIdentifier = append(m.Mapping.RecordIdentifier, "")
		}
	}
	m.Mapping.RecordIdentifier[1] = signatureOriginal
	return nil
}

func (m *medUBCat) GetSource() string {
	return ""
}

func (m *medUBCat) SetSource(source string) error {
	return nil
}

func (m *medUBCat) GetTitle() *multilangString.MultiLangString {
	mls := &multilangString.MultiLangString{}
	mls.Set(m.cast().GetMainTitle())
	return mls
}

func (m *medUBCat) SetTitle(title *multilangString.MultiLangString) error {
	if err := m.cast().SetMainTitle(title.String().String()); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *medUBCat) GetSeries() string {
	return m.cast().GetSeriesTitle()
}

func (m *medUBCat) SetSeries(series string) error {
	if err := m.cast().SetSeriesTitle(series); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *medUBCat) GetPlace() string {
	return m.cast().GetPublicationPlace()
}

func (m *medUBCat) SetPlace(place string) error {
	if err := m.cast().SetPublicationPlace(place); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *medUBCat) GetDate() string {
	return m.cast().GetPublicationDate()
}

func (m *medUBCat) SetDate(date string) error {
	if err := m.cast().SetPublicationDate(date); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *medUBCat) GetCollectionTitle() string {
	return m.cast().GetHostTitle()
}

func (m *medUBCat) SetCollectionTitle(collectionTitle string) error {
	if err := m.cast().SetHostTitle(collectionTitle); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *medUBCat) GetPersons() []sourcetype.Person {
	var persons []sourcetype.Person
	pMap := m.cast().GetPersons()
	for role, pList := range pMap {
		for _, p := range pList {
			persons = append(persons, sourcetype.Person{
				Name: p.Name,
				Role: role,
			})
		}
	}
	return persons
}

func (m *medUBCat) SetPersons(persons []sourcetype.Person) error {
	pMap := make(map[string][]ubschema.ResultPerson)
	for _, p := range persons {
		role := p.Role
		if role == "" {
			role = "author"
		}
		pMap[role] = append(pMap[role], ubschema.ResultPerson{
			Name: p.Name,
		})
	}
	if err := m.cast().SetPersons(pMap); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *medUBCat) AddPerson(p sourcetype.Person) {
	persons := m.GetPersons()
	persons = append(persons, p)
	m.SetPersons(persons)
}

func (m *medUBCat) GetACL() map[string][]string {
	acl := make(map[string][]string)
	if m.ACL != nil {
		acl["content"] = m.ACL.Content
		acl["meta"] = m.ACL.Meta
		acl["preview"] = m.ACL.Preview
	}
	return acl
}

func (m *medUBCat) SetACL(acl map[string][]string) error {
	if m.ACL == nil {
		m.ACL = &ubschema.ACL{}
	}
	if val, ok := acl["content"]; ok {
		m.ACL.Content = val
	}
	if val, ok := acl["meta"]; ok {
		m.ACL.Meta = val
	}
	if val, ok := acl["preview"]; ok {
		m.ACL.Preview = val
	}
	return nil
}

func (m *medUBCat) GetCatalog() []string {
	return nil
}

func (m *medUBCat) SetCatalog(catalog []string) error {
	return nil
}

func (m *medUBCat) GetCategory() []string {
	return nil
}

func (m *medUBCat) SetCategory(category []string) error {
	return nil
}

func (m *medUBCat) GetTags() []string {
	return m.Flags
}

func (m *medUBCat) SetTags(tags []string) error {
	m.Flags = tags
	return nil
}

func (m *medUBCat) GetMedia() map[string]sourcetype.MediaList {
	mediaMap := make(map[string]sourcetype.MediaList)
	if m.Mapping != nil {
		for _, files := range m.Mapping.Files {
			if files.Media != nil {
				for _, med := range files.Media.Medias {
					kind := med.Type
					if kind == "" {
						kind = "unknown"
					}
					mediaItem := sourcetype.Media{
						Name:     med.FileName,
						Mimetype: med.Mimetype,
						Type:     med.Type,
						Uri:      med.Uri,
						Width:    med.Width,
						Height:   med.Height,
						Duration: med.Duration,
					}
					mediaMap[kind] = append(mediaMap[kind], mediaItem)
				}
			}
		}
	}
	return mediaMap
}

func (m *medUBCat) SetMedia(media map[string]sourcetype.MediaList) error {
	return nil
}

func (m *medUBCat) AddMedia(kind string, m2 sourcetype.Media) {
}

func (m *medUBCat) GetPoster() *sourcetype.Media {
	if m.Mapping != nil {
		for _, files := range m.Mapping.Files {
			if files.Media != nil && files.Media.Poster != nil {
				return &sourcetype.Media{
					Name:     files.Media.Poster.FileName,
					Mimetype: files.Media.Poster.Mimetype,
					Type:     files.Media.Poster.Type,
					Uri:      files.Media.Poster.Uri,
					Width:    files.Media.Poster.Width,
					Height:   files.Media.Poster.Height,
					Duration: files.Media.Poster.Duration,
				}
			}
		}
	}
	return &sourcetype.Media{}
}

func (m *medUBCat) SetPoster(poster *sourcetype.Media) error {
	return nil
}

func (m *medUBCat) GetNotes() []sourcetype.Note {
	var notes []sourcetype.Note
	note := m.cast().GetGeneralNote()
	if note != "" {
		notes = append(notes, sourcetype.Note{
			Title: "General",
			Note:  template.HTML(note),
		})
	}
	return notes
}

func (m *medUBCat) SetNotes(notes []sourcetype.Note) error {
	for _, n := range notes {
		if err := m.cast().SetGeneralNote(string(n.Note)); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func (m *medUBCat) GetUrl() string {
	return ""
}

func (m *medUBCat) SetUrl(url string) error {
	return nil
}

func (m *medUBCat) GetAbstract() *multilangString.MultiLangString {
	mls := &multilangString.MultiLangString{}
	abs := m.cast().GetAbstract()
	if abs != nil {
		mls.Set(abs.String())
	}
	return mls
}

func (m *medUBCat) SetAbstract(abstract *multilangString.MultiLangString) error {
	if err := m.cast().SetAbstract(metaString.NewMetaString(abstract.String().String())); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *medUBCat) GetReferences() []sourcetype.Reference {
	return nil
}

func (m *medUBCat) SetReferences(references []sourcetype.Reference) error {
	return nil
}

func (m *medUBCat) GetMeta() *sourcetype.Metalist {
	return &sourcetype.Metalist{}
}

func (m *medUBCat) SetMeta(meta *sourcetype.Metalist) error {
	return nil
}

func (m *medUBCat) AddMeta(key, value string) {
}

func (m *medUBCat) GetExtra() *sourcetype.Metalist {
	return &sourcetype.Metalist{}
}

func (m *medUBCat) SetExtra(extra *sourcetype.Metalist) error {
	return nil
}

func (m *medUBCat) AddExtra(key, value string) {
}

func (m *medUBCat) GetVars() *sourcetype.Varlist {
	return &sourcetype.Varlist{}
}

func (m *medUBCat) SetVars(vars *sourcetype.Varlist) error {
	return nil
}

func (m *medUBCat) AddVar(key string, value []string) {
}

func (m *medUBCat) GetType() string {
	return m.Type
}

func (m *medUBCat) SetType(t string) error {
	m.Type = t
	return nil
}

func (m *medUBCat) GetQueries() []sourcetype.Query {
	return nil
}

func (m *medUBCat) SetQueries(queries []sourcetype.Query) error {
	return nil
}

func (m *medUBCat) GetContentStr() string {
	return ""
}

func (m *medUBCat) SetContentStr(contentStr string) error {
	return nil
}

func (m *medUBCat) GetContentMime() string {
	return ""
}

func (m *medUBCat) SetContentMime(contentMime string) error {
	return nil
}

func (m *medUBCat) GetHasMedia() bool {
	return len(m.GetMedia()) > 0
}

func (m *medUBCat) SetHasMedia(hasMedia bool) error {
	return nil
}

func (m *medUBCat) GetMediatype() []string {
	media := m.GetMedia()
	var types []string
	for t := range media {
		types = append(types, t)
	}
	return types
}

func (m *medUBCat) SetMediatype(mediatype []string) error {
	return nil
}

func (m *medUBCat) GetDateAdded() time.Time {
	return time.Time{}
}

func (m *medUBCat) SetDateAdded(dateAdded time.Time) error {
	return nil
}

func (m *medUBCat) GetTimestamp() time.Time {
	return m.Timestamp
}

func (m *medUBCat) SetTimestamp(timestamp time.Time) error {
	m.Timestamp = timestamp
	return nil
}

func (m *medUBCat) GetPublisher() string {
	return m.cast().GetPublicationPublisher()
}

func (m *medUBCat) SetPublisher(publisher string) error {
	if err := m.cast().SetPublicationPublisher(publisher); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *medUBCat) GetRights() string {
	return ""
}

func (m *medUBCat) SetRights(rights string) error {
	return nil
}

func (m *medUBCat) GetLicense() string {
	return ""
}

func (m *medUBCat) SetLicense(license string) error {
	return nil
}
