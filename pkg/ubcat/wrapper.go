package ubcat

import (
	"html/template"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/je4/revcat/v2/pkg/sourcetype"
	ubschema "gitlab.switch.ch/ub-unibas/rdv2/ubcat/v2/pkg/schema"
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
	recordIdentifiers := m.cast().GetRecordIdentifier()
	for _, recordIdentifier := range recordIdentifiers {
		if strings.HasPrefix(recordIdentifier, "original:") {
			continue
		}
		return recordIdentifier
	}
	return ""
}

func (m *medUBCat) SetSignature(signature string) error {
	return errors.WithStack(m.cast().SetRecordIdentifier(append([]string{signature}, m.cast().GetRecordIdentifier()...)))
}

func (m *medUBCat) GetSignatureOriginal() string {
	recordIdentifiers := m.cast().GetRecordIdentifier()
	for _, recordIdentifier := range recordIdentifiers {
		if strings.HasPrefix(recordIdentifier, "original:") {
			return recordIdentifier[len("original:"):]
		}
	}
	return ""
}

func (m *medUBCat) SetSignatureOriginal(signatureOriginal string) error {
	return errors.WithStack(m.cast().SetRecordIdentifier(append([]string{"original:" + signatureOriginal}, m.cast().GetRecordIdentifier()...)))
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

func (m *medUBCat) AddPerson(p sourcetype.Person) error {
	persons := m.GetPersons()
	persons = append(persons, p)
	if err := m.SetPersons(persons); err != nil {
		return errors.WithStack(err)
	}
	return nil
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
	return m.Sets
}

func (m *medUBCat) SetCatalog(catalog []string) error {
	m.Sets = catalog
	return nil
}

func (m *medUBCat) GetCategory() []string {
	if m.Mapping == nil || m.Mapping.Subject == nil || m.Mapping.Subject.Local == nil {
		return nil
	}
	return m.Mapping.Subject.Local["category"]
}

func (m *medUBCat) SetCategory(category []string) error {
	if m.Mapping == nil {
		m.Mapping = &ubschema.Mapping001{}
	}
	if m.Mapping.Subject == nil {
		m.Mapping.Subject = &ubschema.Subject{}
	}
	if m.Mapping.Subject.Local == nil {
		m.Mapping.Subject.Local = make(map[string][]string)
	}
	m.Mapping.Subject.Local["category"] = category
	return nil
}

func (m *medUBCat) GetTags() []string {
	if m.Mapping == nil || m.Mapping.Subject == nil || m.Mapping.Subject.Local == nil {
		return nil
	}
	return m.Mapping.Subject.Local["tag"]
}

func (m *medUBCat) SetTags(tags []string) error {
	if m.Mapping == nil {
		m.Mapping = &ubschema.Mapping001{}
	}
	if m.Mapping.Subject == nil {
		m.Mapping.Subject = &ubschema.Subject{}
	}
	if m.Mapping.Subject.Local == nil {
		m.Mapping.Subject.Local = make(map[string][]string)
	}
	m.Mapping.Subject.Local["tag"] = tags
	return nil
}

func (m *medUBCat) GetMedia() map[string]sourcetype.MediaList {
	mediaMap := make(map[string]sourcetype.MediaList)
	medias := m.cast().GetMedia()
	for _, med := range medias {
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
	return mediaMap
}

func (m *medUBCat) SetMedia(media map[string]sourcetype.MediaList) error {
	var medias []*ubschema.Medias
	for _, list := range media {
		for _, med := range list {
			medias = append(medias, &ubschema.Medias{
				FileName: med.Name,
				Mimetype: med.Mimetype,
				Type:     med.Type,
				Uri:      med.Uri,
				Width:    med.Width,
				Height:   med.Height,
				Duration: med.Duration,
			})
		}
	}
	if err := m.cast().SetMedia(medias); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *medUBCat) AddMedia(kind string, m2 sourcetype.Media) error {
	if err := m.cast().AddMedia(&ubschema.Medias{
		FileName: m2.Name,
		Mimetype: m2.Mimetype,
		Type:     m2.Type,
		Uri:      m2.Uri,
		Width:    m2.Width,
		Height:   m2.Height,
		Duration: m2.Duration,
	}); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *medUBCat) GetPoster() *sourcetype.Media {
	poster := m.cast().GetPoster()
	if poster == nil {
		return &sourcetype.Media{}
	}
	return &sourcetype.Media{
		Name:     poster.FileName,
		Mimetype: poster.Mimetype,
		Type:     poster.Type,
		Uri:      poster.Uri,
		Width:    poster.Width,
		Height:   poster.Height,
		Duration: poster.Duration,
	}
}

func (m *medUBCat) SetPoster(poster *sourcetype.Media) error {
	if poster == nil {
		return nil
	}
	if err := m.cast().SetPoster(&ubschema.Medias{
		FileName: poster.Name,
		Mimetype: poster.Mimetype,
		Type:     poster.Type,
		Uri:      poster.Uri,
		Width:    poster.Width,
		Height:   poster.Height,
		Duration: poster.Duration,
	}); err != nil {
		return errors.WithStack(err)
	}
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
	abs := m.cast().GetAbstracts()
	mls := &multilangString.MultiLangString{}
	for _, l := range abs {
		mls.Set(l.String())
	}
	return mls
}

func (m *medUBCat) SetAbstract(abstract *multilangString.MultiLangString) error {
	if abstract == nil {
		return errors.New("abstract is nil")
	}
	if err := m.cast().SetAbstract(*abstract...); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (m *medUBCat) GetReferences() []sourcetype.Reference {
	if m.Mapping == nil || m.Mapping.RelatedItem == nil || m.Mapping.RelatedItem.Other == nil {
		return nil
	}
	var refs []sourcetype.Reference
	for _, other := range m.Mapping.RelatedItem.Other {
		refs = append(refs, sourcetype.Reference{
			Type:      other.DisplayConstant,
			Title:     other.Title,
			Signature: other.InternalIdentifier,
		})
	}
	return refs
}

func (m *medUBCat) SetReferences(references []sourcetype.Reference) error {
	if m.Mapping == nil {
		m.Mapping = &ubschema.Mapping001{}
	}
	if m.Mapping.RelatedItem == nil {
		m.Mapping.RelatedItem = &ubschema.RelatedItem{}
	}
	var others []*ubschema.Related
	for _, ref := range references {
		others = append(others, &ubschema.Related{
			DisplayConstant:    ref.Type,
			Title:              ref.Title,
			InternalIdentifier: ref.Signature,
		})
	}
	m.Mapping.RelatedItem.Other = others
	return nil
}

func (m *medUBCat) GetMeta() *sourcetype.Metalist {
	if m.RawJson == nil {
		return &sourcetype.Metalist{}
	}
	meta, ok := m.RawJson["meta"]
	if !ok {
		return &sourcetype.Metalist{}
	}
	metaMap, ok := meta.(map[string]any)
	if !ok {
		return &sourcetype.Metalist{}
	}
	ml := make(sourcetype.Metalist)
	for k, v := range metaMap {
		if s, ok := v.(string); ok {
			ml[k] = s
		}
	}
	return &ml
}

func (m *medUBCat) SetMeta(meta *sourcetype.Metalist) error {
	if meta == nil {
		if m.RawJson != nil {
			delete(m.RawJson, "meta")
		}
		return nil
	}
	if m.RawJson == nil {
		m.RawJson = make(map[string]any)
	}
	metaMap := make(map[string]any)
	for k, v := range *meta {
		metaMap[k] = v
	}
	m.RawJson["meta"] = metaMap
	return nil
}

func (m *medUBCat) AddMeta(key, value string) error {
	meta := m.GetMeta()
	(*meta)[key] = value
	return m.SetMeta(meta)
}

func (m *medUBCat) GetExtra() *sourcetype.Metalist {
	if m.RawJson == nil {
		return &sourcetype.Metalist{}
	}
	extra, ok := m.RawJson["extra"]
	if !ok {
		return &sourcetype.Metalist{}
	}
	extraMap, ok := extra.(map[string]any)
	if !ok {
		return &sourcetype.Metalist{}
	}
	ml := make(sourcetype.Metalist)
	for k, v := range extraMap {
		if s, ok := v.(string); ok {
			ml[k] = s
		}
	}
	return &ml
}

func (m *medUBCat) SetExtra(extra *sourcetype.Metalist) error {
	if extra == nil {
		if m.RawJson != nil {
			delete(m.RawJson, "extra")
		}
		return nil
	}
	if m.RawJson == nil {
		m.RawJson = make(map[string]any)
	}
	extraMap := make(map[string]any)
	for k, v := range *extra {
		extraMap[k] = v
	}
	m.RawJson["extra"] = extraMap
	return nil
}

func (m *medUBCat) AddExtra(key, value string) error {
	extra := m.GetExtra()
	(*extra)[key] = value
	return m.SetExtra(extra)
}

func (m *medUBCat) GetVars() *sourcetype.Varlist {
	if m.RawJson == nil {
		return &sourcetype.Varlist{}
	}
	additional, ok := m.RawJson["additional"]
	if !ok {
		return &sourcetype.Varlist{}
	}
	addMap, ok := additional.(map[string]any)
	if !ok {
		return &sourcetype.Varlist{}
	}
	vl := make(sourcetype.Varlist)
	for k, v := range addMap {
		if s, ok := v.(string); ok {
			vl[k] = []string{s}
		} else if sl, ok := v.([]any); ok {
			var strList []string
			for _, item := range sl {
				if s, ok := item.(string); ok {
					strList = append(strList, s)
				}
			}
			vl[k] = strList
		} else if sl, ok := v.([]string); ok {
			vl[k] = sl
		}
	}
	return &vl
}

func (m *medUBCat) SetVars(vars *sourcetype.Varlist) error {
	if vars == nil {
		if m.RawJson != nil {
			delete(m.RawJson, "additional")
		}
		return nil
	}
	if m.RawJson == nil {
		m.RawJson = make(map[string]any)
	}
	addMap := make(map[string]any)
	for k, v := range *vars {
		if len(v) == 1 {
			addMap[k] = v[0]
		} else if len(v) > 1 {
			addMap[k] = v
		}
	}
	m.RawJson["additional"] = addMap
	return nil
}

func (m *medUBCat) AddVar(key string, value []string) error {
	vars := m.GetVars()
	vars.Append(key, value)
	return m.SetVars(vars)
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
