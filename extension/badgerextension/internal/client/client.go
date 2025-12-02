package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"go.opentelemetry.io/collector/extension/xextension/storage"
)

type Client interface {
	storage.Client
	RunValueLogGC(discardRatio float64) error
}

type ClientOptions interface {
	Apply(c *client)
}

var _ Client = (*client)(nil)

type client struct {
	db *badger.DB
}

func NewClient(path string, opts ...ClientOptions) (Client, error) {
	c := &client{}
	for _, opt := range opts {
		opt.Apply(c)
	}

	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	c.db = db

	return c, nil
}

// Get will retrieve data from storage that corresponds to the specified key
func (c *client) Get(ctx context.Context, key string) ([]byte, error) {
	tx := c.db.NewTransaction(false)
	defer tx.Discard()

	item, err := tx.Get([]byte(key))
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	val, err := item.ValueCopy(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to copy value: %w", err)
	}
	return val, nil
}

// Set will store data. The data can be retrieved using the same key
func (c *client) Set(ctx context.Context, key string, value []byte) error {
	tx := c.db.NewTransaction(true)
	defer tx.Discard()

	err := tx.Set([]byte(key), value)
	if err != nil {
		return fmt.Errorf("failed to set item: %w", err)
	}
	return tx.Commit()
}

func (c *client) Delete(ctx context.Context, key string) error {
	tx := c.db.NewTransaction(true)
	defer tx.Discard()

	err := tx.Delete([]byte(key))
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}
	return tx.Commit()
}

func (c *client) Batch(ctx context.Context, ops ...*storage.Operation) error {
	var wb *badger.WriteBatch
	for _, op := range ops {
		writes := false
		if op.Type == storage.Set || op.Type == storage.Delete {
			writes = true
		}

		if writes && wb == nil {
			wb = c.db.NewWriteBatch()
			defer wb.Cancel()
		}

		var err error
		var value []byte
		switch op.Type {
		case storage.Set:
			err = wb.Set([]byte(op.Key), op.Value)
		case storage.Delete:
			err = wb.Delete([]byte(op.Key))
		case storage.Get:
			value, err = c.Get(ctx, op.Key)
			if err == nil {
				op.Value = value
			}
		default:
			return errors.New("wrong operation type")
		}
		if err != nil {
			return fmt.Errorf("failed to perform %s on %s item: %w", typeString(op.Type), op.Key, err)
		}
	}

	if wb != nil {
		if err := wb.Flush(); err != nil {
			return fmt.Errorf("failed to flush write batch: %w", err)
		}
	}

	return nil
}

func (c *client) Close(_ context.Context) error {
	return c.db.Close()
}

func (c *client) RunValueLogGC(discardRatio float64) error {
	return c.db.RunValueLogGC(discardRatio)
}

func typeString(t storage.OpType) string {
	switch t {
	case storage.Set:
		return "set"
	case storage.Delete:
		return "delete"
	case storage.Get:
		return "get"
	}
	return "unknown"
}
