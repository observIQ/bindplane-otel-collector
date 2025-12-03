package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/cockroachdb/pebble"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
)

type Client interface {
	storage.Client
}

type client struct {
	db *pebble.DB

	started bool
	sync    bool
}

type ClientOptions struct {
	Sync bool
}

func NewClient(path string, options *ClientOptions) (Client, error) {
	c := &client{
		sync: options.Sync,
	}

	db, err := pebble.Open(path, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	c.db = db
	return c, nil
}

func (c *client) Get(ctx context.Context, key string) ([]byte, error) {
	val, closer, err := c.db.Get([]byte(key))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting key %s: %w", key, err)
	}
	defer closer.Close()

	if val == nil {
		return nil, nil
	}

	return val, nil
}

func (c *client) Set(ctx context.Context, key string, value []byte) error {
	err := c.db.Set([]byte(key), value, &pebble.WriteOptions{})
	if err != nil {
		return fmt.Errorf("error setting key %s: %w", key, err)
	}
	return nil
}

func (c *client) Delete(ctx context.Context, key string) error {
	err := c.db.Delete([]byte(key), &pebble.WriteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting key %s: %w", key, err)
	}
	return nil
}

func (c *client) Batch(ctx context.Context, ops ...*storage.Operation) error {

	wb := c.db.NewBatch()
	defer wb.Close()

	for _, op := range ops {
		var writes bool
		switch op.Type {
		case storage.Set, storage.Delete:
			writes = true
		}

		if writes && wb == nil {
			wb = c.db.NewBatch()
		}

		var err error
		switch op.Type {
		case storage.Set:
			err = wb.Set([]byte(op.Key), op.Value, &pebble.WriteOptions{
				Sync: c.sync,
			})
		case storage.Delete:
			err = wb.Delete([]byte(op.Key), &pebble.WriteOptions{
				Sync: c.sync,
			})
		case storage.Get:
			value, err := c.Get(ctx, op.Key)
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

	return wb.Commit(&pebble.WriteOptions{
		Sync: c.sync,
	})
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

func (c *client) Start(_tx context.Context, _ component.Host) error {
	c.started = true
	return nil
}

func (c *client) Close(_ context.Context) error {
	// if c.started && c.db != nil {
	// 	return c.db.Close()
	// }
	return nil
}
