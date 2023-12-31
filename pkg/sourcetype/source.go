package sourcetype

import (
	"bytes"
	"compress/gzip"
	"github.com/je4/zsearch/v2/pkg/translate"

	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"html/template"
	"io"
	"time"
)

type SourceData struct {
	ID                string                     `json:"-"`
	Signature         string                     `json:"signature"`
	SignatureOriginal string                     `json:"signatureoriginal"`
	Source            string                     `json:"source"`
	Title             *translate.MultiLangString `json:"title"`
	Series            string                     `json:"series"`
	Place             string                     `json:"place"`
	Date              string                     `json:"date"`
	CollectionTitle   string                     `json:"collectiontitle"`
	Persons           []Person                   `json:"persons"`
	ACL               map[string][]string        `json:"acl"`
	Catalog           []string                   `json:"catalog"`
	Category          []string                   `json:"category"`
	Tags              []string                   `json:"tags"`
	Media             map[string]MediaList       `json:"media"`
	Poster            *Media                     `json:"poster"`
	Notes             []Note                     `json:"notes"`
	Url               string                     `json:"url"`
	Abstract          *translate.MultiLangString `json:"abstract"`
	References        []Reference                `json:"references"`
	Meta              *Metalist                  `json:"meta,omitempty"`
	Extra             *Metalist                  `json:"extra,omitempty"`
	Vars              *Varlist                   `json:"vars,omitempty"`
	Type              string                     `json:"type"`
	Queries           []Query                    `json:"queries,omitempty"`
	ContentStr        string                     `json:"-"`
	ContentMime       string                     `json:"-"`
	HasMedia          bool                       `json:"hasmedia"`
	Mediatype         []string                   `json:"mediatype"`
	DateAdded         time.Time                  `json:"dateadded"`
	Timestamp         time.Time                  `json:"timestamp"`
	Publisher         string                     `json:"publisher"`
	Rights            string                     `json:"rights"`
	License           string                     `json:"license"`
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

type Person struct {
	Name string `json:"name"`
	Role string `json:"role"`
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

type Metalist map[string]string

func (ml *Metalist) UnmarshalJSON(b []byte) error {
	type kv struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	var arr []kv

	m := Metalist{}
	if err := json.Unmarshal(b, &arr); err != nil {
		return err
	}
	for _, val := range arr {
		m[val.Key] = val.Value
	}
	*ml = m
	return nil
}

func (ml Metalist) MarshalJSON() ([]byte, error) {
	type kv struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	var arr []kv
	for key, val := range ml {
		arr = append(arr, kv{Key: key, Value: val})
	}
	return json.Marshal(arr)
}

type Varlist map[string][]string

func (vl *Varlist) UnmarshalJSON(b []byte) error {
	type kv struct {
		Key   string   `json:"key"`
		Value []string `json:"value"`
	}
	var arr []kv

	m := Varlist{}
	if err := json.Unmarshal(b, &arr); err != nil {
		return err
	}
	for _, val := range arr {
		m[val.Key] = val.Value
	}
	*vl = m
	return nil
}

func (vl Varlist) MarshalJSON() ([]byte, error) {
	type kv struct {
		Key   string   `json:"key"`
		Value []string `json:"value"`
	}
	var arr []kv
	for key, val := range vl {
		arr = append(arr, kv{Key: key, Value: val})
	}
	return json.Marshal(arr)
}

func (vl Varlist) Append(key string, values []string) {
	if _, ok := vl[key]; !ok {
		vl[key] = []string{}
	}
	vl[key] = append(vl[key], values...)
}

func (vl Varlist) AppendMap(mv map[string][]string) {
	for key, values := range mv {
		vl.Append(key, values)
	}
}

func (vl Varlist) Unique() *Varlist {
	// todo: optimize it
	unique := func(arr []string) []string {
		occured := map[string]bool{}
		result := []string{}
		for e := range arr {
			// check if already the mapped
			// variable is set to true or not
			if occured[arr[e]] != true {
				occured[arr[e]] = true
				// Append to result slice.
				result = append(result, arr[e])
			}
		}

		return result
	}
	result := Varlist{}
	for key, values := range vl {
		result.Append(key, unique(values))
	}
	return &result
}
