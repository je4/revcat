package graph

import (
	"context"
	emperror "emperror.dev/errors"
	"encoding/json"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/revcat/v2/tools/graph/model"
)

func loadEntries(ctx context.Context, elastic *elasticsearch.TypedClient, index string, signatures []string) ([]sourcetype.SourceData, error) {
	result, err := elastic.Mget().Index(index).Ids(signatures...).Do(ctx)
	if err != nil {
		return nil, emperror.Wrapf(err, "cannot load '%s' entries %v", index, signatures)
	}
	var docs = make([]sourcetype.SourceData, 0)
	for _, docInt := range result.Docs {
		doc, ok := docInt.(map[string]interface{})
		if !ok {
			return nil, emperror.Errorf("cannot convert doc %v to map", docInt)
		}
		if found, ok := doc["found"].(bool); !ok || !found {
			return nil, emperror.Errorf("document %s not found", doc["id"])
		}
		id, ok := doc["_id"].(string)
		if !ok {
			return nil, emperror.Errorf("cannot convert doc id %v to string", doc["_id"])
		}
		sourceMap, ok := doc["_source"].(map[string]interface{})
		if !ok {
			return nil, emperror.Errorf("cannot convert doc source %v to map", doc["_source"])
		}
		jsonBytes, err := json.Marshal(sourceMap)
		if err != nil {
			return nil, emperror.Wrapf(err, "cannot marshal source %v", sourceMap)
		}
		source := sourcetype.SourceData{ID: id}
		if err := json.Unmarshal(jsonBytes, &source); err != nil {
			return nil, emperror.Wrapf(err, "cannot unmarshal source %v", source)
		}
		docs = append(docs, source)
	}
	return docs, nil
}

func sourceMediaToMedia(m *sourcetype.Media) *model.Media {
	if m == nil {
		return nil
	}
	media := &model.Media{
		Name:     m.Name,
		Mimetype: m.Mimetype,
		//		Pronom:      &m.,
		Type:        m.Type,
		URI:         m.Uri,
		Orientation: int(m.Orientation),
		Fulltext:    &m.Fulltext,
		Width:       int(m.Width),
		Height:      int(m.Height),
		Length:      int(m.Duration),
	}
	return media
}

func sourceToMediathekBaseEntry(src *sourcetype.SourceData) *model.MediathekBaseEntry {
	entry := &model.MediathekBaseEntry{
		ID:                src.ID,
		Signature:         src.Signature,
		SignatureOriginal: src.SignatureOriginal,
		Source:            src.Source,
		Title:             src.Title,
		Series:            &src.Series,
		Place:             &src.Place,
		Date:              &src.Date,
		Category:          src.Category,
		Tags:              src.Tags,
		URL:               &src.Url,
		Publisher:         &src.Publisher,
		Rights:            &src.Rights,
		License:           &src.License,
		Type:              &src.Type,
		References:        make([]*model.Reference, 0),
		Poster:            sourceMediaToMedia(src.Poster),
	}
	return entry
}

func sourceToMediathekFullEntry(src *sourcetype.SourceData) *model.MediathekFullEntry {
	entry := &model.MediathekFullEntry{
		ID:             src.ID,
		Base:           sourceToMediathekBaseEntry(src),
		Notes:          []*model.Note{},
		Abstract:       &src.Abstract,
		ReferencesFull: []*model.MediathekBaseEntry{},
		Extra:          []*model.KeyValue{},
		Media:          []*model.MediaList{},
	}
	for _, note := range src.Notes {
		entry.Notes = append(entry.Notes, &model.Note{
			Title: &note.Title,
			Text:  string(note.Note),
		})
	}
	if src.Extra != nil {
		for key, val := range *src.Extra {
			entry.Extra = append(entry.Extra, &model.KeyValue{
				Key:   key,
				Value: val,
			})
		}
	}
	if src.Media != nil {
		for key, ml := range src.Media {
			mediaList := &model.MediaList{
				Name:  key,
				Items: make([]*model.Media, 0),
			}
			for _, media := range ml {
				mediaList.Items = append(mediaList.Items, sourceMediaToMedia(&media))
			}
			entry.Media = append(entry.Media, mediaList)
		}
	}
	return entry
}
