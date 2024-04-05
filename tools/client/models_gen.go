// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package client

type FacetValue interface {
	IsFacetValue()
}

type Facet struct {
	Name   string       `json:"name"`
	Values []FacetValue `json:"values,omitempty"`
}

type FacetValueInt struct {
	IntVal int64 `json:"intVal"`
	Count  int64 `json:"count"`
}

func (FacetValueInt) IsFacetValue() {}

type FacetValueString struct {
	StrVal string `json:"strVal"`
	Count  int64  `json:"count"`
}

func (FacetValueString) IsFacetValue() {}

type InFacet struct {
	Term  *InFacetTerm `json:"term,omitempty"`
	Query *InFilter    `json:"query"`
}

type InFacetTerm struct {
	Field       string   `json:"field"`
	Name        string   `json:"name"`
	MinDocCount int64    `json:"minDocCount"`
	Size        int64    `json:"size"`
	Include     []string `json:"include,omitempty"`
	Exclude     []string `json:"exclude,omitempty"`
}

type InFilter struct {
	BoolTerm *InFilterBoolTerm `json:"boolTerm,omitempty"`
}

type InFilterBoolTerm struct {
	Field  string   `json:"field"`
	And    bool     `json:"and"`
	Values []string `json:"values,omitempty"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Media struct {
	Name        string  `json:"name"`
	Mimetype    string  `json:"mimetype"`
	Pronom      *string `json:"pronom,omitempty"`
	Type        string  `json:"type"`
	URI         string  `json:"uri"`
	Orientation int64   `json:"orientation"`
	Fulltext    *string `json:"fulltext,omitempty"`
	Width       int64   `json:"width"`
	Height      int64   `json:"height"`
	Length      int64   `json:"length"`
}

type MediaList struct {
	Name  string   `json:"name"`
	Items []*Media `json:"items"`
}

type MediathekBaseEntry struct {
	ID                string             `json:"id"`
	Signature         string             `json:"signature"`
	SignatureOriginal string             `json:"signatureOriginal"`
	Source            string             `json:"source"`
	Title             []*MultiLangString `json:"title"`
	Series            *string            `json:"series,omitempty"`
	Place             *string            `json:"place,omitempty"`
	Date              *string            `json:"date,omitempty"`
	CollectionTitle   *string            `json:"collectionTitle,omitempty"`
	Person            []*Person          `json:"person,omitempty"`
	Catalog           []string           `json:"catalog,omitempty"`
	Category          []string           `json:"category,omitempty"`
	Tags              []string           `json:"tags,omitempty"`
	URL               *string            `json:"url,omitempty"`
	Publisher         *string            `json:"publisher,omitempty"`
	Rights            *string            `json:"rights,omitempty"`
	License           *string            `json:"license,omitempty"`
	References        []*Reference       `json:"references,omitempty"`
	Type              *string            `json:"type,omitempty"`
	Poster            *Media             `json:"poster,omitempty"`
}

type MediathekFullEntry struct {
	ID             string                `json:"id"`
	Base           *MediathekBaseEntry   `json:"base"`
	Notes          []*Note               `json:"notes,omitempty"`
	Abstract       []*MultiLangString    `json:"abstract,omitempty"`
	ReferencesFull []*MediathekBaseEntry `json:"referencesFull,omitempty"`
	Extra          []*KeyValue           `json:"extra,omitempty"`
	Media          []*MediaList          `json:"media,omitempty"`
}

type MultiLangString struct {
	Lang       string `json:"lang"`
	Value      string `json:"value"`
	Translated bool   `json:"translated"`
}

type Note struct {
	Title *string `json:"title,omitempty"`
	Text  string  `json:"text"`
}

type PageInfo struct {
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
	CurrentCursor   string `json:"currentCursor"`
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
}

type Person struct {
	Name string  `json:"name"`
	Role *string `json:"role,omitempty"`
}

type Query struct {
}

type Reference struct {
	Type      *string `json:"type,omitempty"`
	Title     *string `json:"title,omitempty"`
	Signature string  `json:"signature"`
}

type SearchResult struct {
	TotalCount int64                 `json:"totalCount"`
	PageInfo   *PageInfo             `json:"pageInfo"`
	Edges      []*MediathekFullEntry `json:"edges"`
	Facets     []*Facet              `json:"facets"`
}

type SortField struct {
	Field string `json:"field"`
	Order string `json:"order"`
}
