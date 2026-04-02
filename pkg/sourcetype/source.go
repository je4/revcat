package sourcetype

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"html/template"
	"io"
	"time"

	"github.com/pkg/errors"
	"go.ub.unibas.ch/metastring/pkg/multilangString"
)

type SourceData struct {
	ID                string                           `json:"-"`
	Signature         string                           `json:"signature"`
	SignatureOriginal string                           `json:"signatureoriginal"`
	Source            string                           `json:"source"`
	Title             *multilangString.MultiLangString `json:"title"`
	Series            string                           `json:"series"`
	Place             string                           `json:"place"`
	Date              string                           `json:"date"`
	CollectionTitle   string                           `json:"collectiontitle"`
	Persons           []Person                         `json:"persons"`
	ACL               map[string][]string              `json:"acl"`
	Catalog           []string                         `json:"catalog"`
	Category          []string                         `json:"category"`
	Tags              []string                         `json:"tags"`
	Media             map[string]MediaList             `json:"media"`
	Poster            *Media                           `json:"poster"`
	Notes             []Note                           `json:"notes"`
	Url               string                           `json:"url"`
	Abstract          *multilangString.MultiLangString `json:"abstract"`
	References        []Reference                      `json:"references"`
	Meta              *Metalist                        `json:"meta,omitempty"`
	Extra             *Metalist                        `json:"extra,omitempty"`
	Vars              *Varlist                         `json:"vars,omitempty"`
	Type              string                           `json:"type"`
	Queries           []Query                          `json:"queries,omitempty"`
	ContentStr        string                           `json:"-"`
	ContentMime       string                           `json:"-"`
	HasMedia          bool                             `json:"hasmedia"`
	Mediatype         []string                         `json:"mediatype"`
	DateAdded         time.Time                        `json:"dateadded"`
	Timestamp         time.Time                        `json:"timestamp"`
	Publisher         string                           `json:"publisher"`
	Rights            string                           `json:"rights"`
	License           string                           `json:"license"`
}

func GUnzip(data string) (string, error) {
	var src, dest bytes.Buffer

	bytedata, err := base64.StdEncoding.DecodeString(data)
	if _, err := src.Write(bytedata); err != nil {
		return "", errors.Wrap(err, "cannot write data into buffer")
	}
	zr, err := gzip.NewReader(&src)
	if err != nil {
		return "", errors.Wrap(err, "cannot create gzip reader")
	}
	if _, err := io.Copy(&dest, zr); err != nil {
		return "", errors.Wrap(err, "uncompress data")
	}
	return dest.String(), nil
}

type Identifier struct {
	ID         string `json:"id"`
	URL        string `json:"url,omitempty"`
	Additional string `json:"additional,omitempty"`
}

type Person struct {
	Name             string                `json:"name"`
	Role             string                `json:"role"`
	AlternativeNames []string              `json:"alternative_names,omitempty"`
	Year             int                   `json:"year,omitempty"`
	Web              []string              `json:"web,omitempty"`
	Identifier       map[string]Identifier `json:"identifier"`
}

type Media struct {
	Name        string `json:"name"`
	Mimetype    string `json:"mimetype"`
	Type        string `json:"type"`
	Uri         string `json:"uri"`
	Width       int64  `json:"width,omitempty"`
	Height      int64  `json:"height,omitempty"`
	Orientation int64  `json:"orientation,omitempty"`
	Duration    int64  `json:"duration,omitempty"`
	Fulltext    string `json:"fulltext,omitempty"`
}

type Query struct {
	Label  string `json:"label"`
	Search string `json:"search"`
}

type MediaList []Media

func (ml MediaList) Len() int           { return len(ml) }
func (ml MediaList) Swap(i, j int)      { ml[i], ml[j] = ml[j], ml[i] }
func (ml MediaList) Less(i, j int) bool { return ml[i].Name < ml[j].Name }

type Note struct {
	Title string        `json:"title"`
	Note  template.HTML `json:"note"`
}

type Reference struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Signature string `json:"signature"`
}
