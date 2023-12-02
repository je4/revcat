// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

type Facet struct {
	Field  string        `json:"field"`
	Values []*FacetValue `json:"values"`
}

type FacetInput struct {
	Name         string   `json:"name"`
	Field        string   `json:"field"`
	ValuesString []string `json:"valuesString,omitempty"`
	ValuesInt    []int    `json:"ValuesInt,omitempty"`
}

type FacetValue struct {
	ValStr *string `json:"valStr,omitempty"`
	ValInt *int    `json:"valInt,omitempty"`
	Count  int     `json:"count"`
}

type FilterInput struct {
	Field        string   `json:"field"`
	ValuesString []string `json:"valuesString,omitempty"`
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
	Orientation int     `json:"orientation"`
	Fulltext    *string `json:"fulltext,omitempty"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Length      int     `json:"length"`
}

type MediaList struct {
	Name  string   `json:"name"`
	Items []*Media `json:"items"`
}

type MediathekBaseEntry struct {
	ID                string       `json:"id"`
	Signature         string       `json:"signature"`
	SignatureOriginal string       `json:"signatureOriginal"`
	Source            string       `json:"source"`
	Title             string       `json:"title"`
	Series            *string      `json:"series,omitempty"`
	Place             *string      `json:"place,omitempty"`
	Date              *string      `json:"date,omitempty"`
	CollectionTitle   *string      `json:"collectionTitle,omitempty"`
	Person            []*Person    `json:"person,omitempty"`
	Catalog           []string     `json:"catalog,omitempty"`
	Category          []string     `json:"category,omitempty"`
	Tags              []string     `json:"tags,omitempty"`
	URL               *string      `json:"url,omitempty"`
	Publisher         *string      `json:"publisher,omitempty"`
	Rights            *string      `json:"rights,omitempty"`
	License           *string      `json:"license,omitempty"`
	References        []*Reference `json:"references,omitempty"`
	Type              *string      `json:"type,omitempty"`
	Poster            *Media       `json:"poster,omitempty"`
}

type MediathekFullEntry struct {
	ID             string                `json:"id"`
	Base           *MediathekBaseEntry   `json:"base"`
	Notes          []*Note               `json:"notes,omitempty"`
	Abstract       *string               `json:"abstract,omitempty"`
	ReferencesFull []*MediathekBaseEntry `json:"referencesFull,omitempty"`
	Extra          []*KeyValue           `json:"extra,omitempty"`
	Media          []*MediaList          `json:"media,omitempty"`
}

type Note struct {
	Title *string `json:"title,omitempty"`
	Text  string  `json:"text"`
}

type PageInfo struct {
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
	StartCursor     string `json:"startCursor"`
	EndCursor       string `json:"endCursor"`
}

type Person struct {
	Name string  `json:"name"`
	Role *string `json:"role,omitempty"`
}

type Reference struct {
	Type      *string `json:"type,omitempty"`
	Title     *string `json:"title,omitempty"`
	Signature string  `json:"signature"`
}

type SearchResult struct {
	TotalCount int                   `json:"totalCount"`
	PageInfo   *PageInfo             `json:"pageInfo"`
	Edges      []*MediathekFullEntry `json:"edges"`
	Facets     []*Facet              `json:"facets"`
}
