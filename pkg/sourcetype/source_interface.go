package sourcetype

import (
	"time"

	"go.ub.unibas.ch/metastring/pkg/multilangString"
)

type Source interface {
	GetID() string
	SetID(id string) error
	GetSignature() string
	SetSignature(signature string) error
	GetSignatureOriginal() string
	SetSignatureOriginal(signatureOriginal string) error
	GetSource() string
	SetSource(source string) error
	GetTitle() *multilangString.MultiLangString
	SetTitle(title *multilangString.MultiLangString) error
	GetSeries() string
	SetSeries(series string) error
	GetPlace() string
	SetPlace(place string) error
	GetDate() string
	SetDate(date string) error
	GetCollectionTitle() string
	SetCollectionTitle(collectionTitle string) error
	GetPersons() []Person
	SetPersons(persons []Person) error
	AddPerson(p Person) error
	GetACL() map[string][]string
	SetACL(acl map[string][]string) error
	GetCatalog() []string
	SetCatalog(catalog []string) error
	GetCategory() []string
	SetCategory(category []string) error
	GetTags() []string
	SetTags(tags []string) error
	GetMedia() map[string]MediaList
	SetMedia(media map[string]MediaList) error
	AddMedia(kind string, m Media) error
	GetPoster() *Media
	SetPoster(poster *Media) error
	GetNotes() []Note
	SetNotes(notes []Note) error
	GetUrl() string
	SetUrl(url string) error
	GetAbstract() *multilangString.MultiLangString
	SetAbstract(abstract *multilangString.MultiLangString) error
	GetReferences() []Reference
	SetReferences(references []Reference) error
	GetMeta() *Metalist
	SetMeta(meta *Metalist) error
	AddMeta(key, value string) error
	GetExtra() *Metalist
	SetExtra(extra *Metalist) error
	AddExtra(key, value string) error
	GetVars() *Varlist
	SetVars(vars *Varlist) error
	AddVar(key string, value []string) error
	GetType() string
	SetType(t string) error
	GetQueries() []Query
	SetQueries(queries []Query) error
	GetContentStr() string
	SetContentStr(contentStr string) error
	GetContentMime() string
	SetContentMime(contentMime string) error
	GetHasMedia() bool
	SetHasMedia(hasMedia bool) error
	GetMediatype() []string
	SetMediatype(mediatype []string) error
	GetDateAdded() time.Time
	SetDateAdded(dateAdded time.Time) error
	GetTimestamp() time.Time
	SetTimestamp(timestamp time.Time) error
	GetPublisher() string
	SetPublisher(publisher string) error
	GetRights() string
	SetRights(rights string) error
	GetLicense() string
	SetLicense(license string) error
}
