package graph

import (
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/revcat/v2/tools/graph/model"
)

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
