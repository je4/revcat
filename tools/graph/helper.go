package graph

import (
	"context"
	"emperror.dev/errors"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/gin-gonic/gin"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/revcat/v2/tools/graph/model"
)

func createFilterQuery(filter *model.InFilter) (types.Query, error) {
	if filter.BoolTerm != nil {
		var query = types.Query{Bool: &types.BoolQuery{}}
		qList := []types.Query{}
		for _, val := range filter.BoolTerm.Values {
			qList = append(qList, types.Query{Term: map[string]types.TermQuery{
				filter.BoolTerm.Field: types.TermQuery{
					Value: val,
				},
			},
			})
		}
		if filter.BoolTerm.And {
			query.Bool.Must = qList
		} else {
			query.Bool.Should = qList
			query.Bool.MinimumShouldMatch = 1
		}
		return query, nil
	}
	return types.Query{}, errors.Errorf("unknown filter type")
}

func stringsFromContext(ctx context.Context, key string) ([]string, error) {
	stringsAny := ctx.Value(key)
	if stringsAny == nil {
		//return nil, errors.Errorf("no value for '%s' found in context", key)
		return []string{}, nil
	}
	strs, ok := stringsAny.([]string)
	if !ok {
		return nil, errors.Errorf("invalid value type for '%s' found in context", key)
	}
	return strs, nil
}

func stringFromContext(ctx context.Context, key string) (string, error) {
	stringAny := ctx.Value(key)
	if stringAny == nil {
		//return nil, errors.Errorf("no value for '%s' found in context", key)
		return "", nil
	}
	str, ok := stringAny.(string)
	if !ok {
		return "", errors.Errorf("invalid value type for '%s' found in context", key)
	}
	return str, nil
}

func ginContextFromContext(ctx context.Context) (*gin.Context, error) {
	ginContext := ctx.Value("GinContextKey")
	if ginContext == nil {
		err := errors.Errorf("could not retrieve gin.Context")
		return nil, err
	}

	gc, ok := ginContext.(*gin.Context)
	if !ok {
		err := errors.Errorf("gin.Context has wrong type")
		return nil, err
	}
	return gc, nil
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
		Title:             []*model.MultiLangString{}, //src.Title,
		Series:            &src.Series,
		Place:             &src.Place,
		Date:              &src.Date,
		Person:            []*model.Person{},
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
	for _, person := range src.Persons {
		p := &model.Person{
			Name: person.Name,
		}
		if person.Role != "" {
			p.Role = &person.Role
		}
		entry.Person = append(entry.Person, p)
	}
	for _, lang := range src.Title.GetNativeLanguages() {
		entry.Title = append(entry.Title, &model.MultiLangString{
			Lang:       lang.String(),
			Value:      src.Title.Get(lang),
			Translated: false,
		})
	}
	for _, lang := range src.Title.GetTranslatedLanguages() {
		entry.Title = append(entry.Title, &model.MultiLangString{
			Lang:       lang.String(),
			Value:      src.Title.Get(lang),
			Translated: true,
		})
	}
	return entry
}

func sourceToMediathekFullEntry(src *sourcetype.SourceData) *model.MediathekFullEntry {
	entry := &model.MediathekFullEntry{
		ID:             src.ID,
		Base:           sourceToMediathekBaseEntry(src),
		Notes:          []*model.Note{},
		Abstract:       []*model.MultiLangString{}, //&src.Abstract,
		ReferencesFull: []*model.MediathekBaseEntry{},
		Extra:          []*model.KeyValue{},
		Media:          []*model.MediaList{},
	}
	for _, lang := range src.Abstract.GetNativeLanguages() {
		entry.Abstract = append(entry.Abstract, &model.MultiLangString{
			Lang:       lang.String(),
			Value:      src.Abstract.Get(lang),
			Translated: false,
		})
	}
	for _, lang := range src.Abstract.GetTranslatedLanguages() {
		entry.Abstract = append(entry.Abstract, &model.MultiLangString{
			Lang:       lang.String(),
			Value:      src.Abstract.Get(lang),
			Translated: true,
		})
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
				Type:  key,
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
