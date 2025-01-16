package resolver

import (
	"bytes"
	"context"
	"emperror.dev/errors"
	"encoding/json"
	"github.com/andybalholm/brotli"
	"github.com/dgraph-io/badger/v4"
	"github.com/je4/revcat/v2/pkg/sourcetype"
	"github.com/je4/revcat/v2/tools/graph/model"
	"github.com/je4/utils/v2/pkg/zLogger"
	"io"
)

func NewBadgerResolver(logger zLogger.ZLogger, db *badger.DB) Resolver {
	return &badgerResolver{
		logger: logger,
		db:     db,
	}
}

type badgerResolver struct {
	logger zLogger.ZLogger
	db     *badger.DB
}

func (b *badgerResolver) loadEntries(ctx context.Context, signatures []string) ([]sourcetype.SourceData, error) {
	var result = []sourcetype.SourceData{}
	if err := b.db.View(func(txn *badger.Txn) error {
		for _, signature := range signatures {
			item, err := txn.Get([]byte(signature))
			if err != nil {
				return errors.Wrapf(err, "cannot get item %v", signature)
			}
			if err := item.Value(func(val []byte) error {
				br := brotli.NewReader(bytes.NewReader(val))
				data, err := io.ReadAll(br)
				if err != nil {
					return errors.Wrapf(err, "cannot read from brotli reader")
				}
				source := sourcetype.SourceData{ID: signature}
				if err := json.Unmarshal(data, &source); err != nil {
					return errors.Wrapf(err, "cannot unmarshal source %v", source)
				}

				result = append(result, source)
				return nil
			}); err != nil {
				return err
			}
			return nil
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (b *badgerResolver) Search(ctx context.Context, query string, facets []*model.InFacet, filter []*model.InFilter, vector []float64, first *int, size *int, cursor *string, sort []*model.SortField) (*model.SearchResult, error) {
	return nil, errors.Errorf("badgerResolver::Search not implemented")
}

func (b *badgerResolver) sourceToMediathekFullEntry(src *sourcetype.SourceData) *model.MediathekFullEntry {
	entry := &model.MediathekFullEntry{
		ID:             src.ID,
		Base:           sourceToMediathekBaseEntry(src),
		Notes:          []*model.Note{},
		Abstract:       []*model.MultiLangString{}, //&src.Abstract,
		ReferencesFull: []*model.MediathekBaseEntry{},
		Extra:          []*model.KeyValue{},
		Media:          []*model.MediaList{},
	}
	var refSignatures = make([]string, 0)
	for _, ref := range src.References {
		if ref.Type == "signature" {
			refSignatures = append(refSignatures, ref.Signature)
		}
	}
	if len(refSignatures) > 0 {
		refs, err := b.loadEntries(context.Background(), refSignatures)
		if err != nil {
			b.logger.Error().Err(err).Msgf("cannot load references %v", refSignatures)
		}
		for _, ref := range refs {
			if ref.Signature == src.Signature {
				// prevent recursion
				continue
			}
			entry.ReferencesFull = append(entry.ReferencesFull, sourceToMediathekBaseEntry(&ref))
		}
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

func (b *badgerResolver) MediathekEntries(ctx context.Context, signatures []string) ([]*model.MediathekFullEntry, error) {
	var result = []*model.MediathekFullEntry{}
	docs, err := b.loadEntries(ctx, signatures)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot load entries %v", signatures)
	}
	for _, doc := range docs {
		result = append(result, b.sourceToMediathekFullEntry(&doc))
	}
	return result, nil
}

func (b *badgerResolver) VectorSearch(ctx context.Context, vector []float64, facets []*model.InFacet, first *int, size *int, cursor *string) (*model.SearchResult, error) {
	return nil, errors.Errorf("badgerResolver::VectorSearch not implemented")
}

func (b *badgerResolver) ReferencesFull(ctx context.Context, obj *model.MediathekFullEntry) ([]*model.MediathekBaseEntry, error) {
	var result = []*model.MediathekBaseEntry{}
	var signatures = []string{}
	for _, ref := range obj.Base.References {
		signatures = append(signatures, ref.Signature)
	}
	docs, err := b.loadEntries(ctx, signatures)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot load entries %v", signatures)
	}
	for _, doc := range docs {
		result = append(result, sourceToMediathekBaseEntry(&doc))
	}
	return result, nil
}

var _ Resolver = (*badgerResolver)(nil)
