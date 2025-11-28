package main

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Document struct {
	ID        string `bson:"_id"`
	CreatedAt int64  `bson:"createdAt"`
	Data      string `bson:"data"`
}

type BatchProcessor struct {
	collection      *mongo.Collection
	batchSize       int64
	manager         *Manager
	createdAtCutoff *int64 // optional cutoff in milliseconds; if nil, no cutoff applied
}

func NewBatchProcessor(coll *mongo.Collection, batchSize int, mgr *Manager) *BatchProcessor {
	if batchSize < 1 {
		batchSize = 100
	}
	return &BatchProcessor{
		collection:      coll,
		batchSize:       int64(batchSize),
		manager:         mgr,
		createdAtCutoff: nil,
	}
}

// WithCreatedAtCutoff sets the createdAt cutoff timestamp in milliseconds
func (bp *BatchProcessor) WithCreatedAtCutoff(cutoff int64) *BatchProcessor {
	bp.createdAtCutoff = &cutoff
	return bp
}

type _claimDoc struct {
	id     string
	result any
}

func (bp *BatchProcessor) Run(ctx context.Context, processFn func(doc Document) any) error {
	jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
	time.Sleep(jitter)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		filter := bson.M{"_claim": bson.M{"$exists": false}}
		if bp.createdAtCutoff != nil {
			filter["createdAt"] = bson.M{"$lt": *bp.createdAtCutoff}
		}

		cursor, err := bp.collection.Find(ctx,
			filter,
			options.Find().SetLimit(bp.batchSize),
		)
		if err != nil {
			return err
		}

		var docs []Document
		if err := cursor.All(ctx, &docs); err != nil {
			cursor.Close(ctx)
			return err
		}
		cursor.Close(ctx)

		if len(docs) == 0 {
			return nil
		}

		results := make(chan _claimDoc, len(docs))
		var wg sync.WaitGroup

		for _, doc := range docs {
			wg.Add(1)
			d := doc
			bp.manager.Submit(func(ctx context.Context) error {
				defer wg.Done()
				result := processFn(d)
				results <- _claimDoc{id: d.ID, result: result}
				return nil
			}, RetryPolicy{MaxAttempts: 1})
		}

		wg.Wait()
		close(results)

		models := make([]mongo.WriteModel, 0, len(docs))
		for r := range results {
			models = append(models, mongo.NewUpdateOneModel().
				SetFilter(bson.M{
					"_id":    r.id,
					"_claim": bson.M{"$exists": false},
				}).
				SetUpdate(bson.M{
					"$set": bson.M{"_claim": r.result},
				}),
			)
		}

		_, err = bp.collection.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
		if err != nil {
			return err
		}
	}
}
