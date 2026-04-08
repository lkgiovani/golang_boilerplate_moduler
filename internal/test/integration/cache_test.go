package integration

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestCacheProvider_SetAndGet(t *testing.T) {
	ctx := context.Background()
	key := "test:set-and-get"
	value := "hello-world"

	err := cacheProvider.Set(ctx, key, value, 10*time.Second)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := cacheProvider.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got != value {
		t.Fatalf("expected %q, got %q", value, got)
	}

	// cleanup
	_ = cacheProvider.Delete(ctx, key)
}

func TestCacheProvider_Delete(t *testing.T) {
	ctx := context.Background()
	key := "test:delete"

	err := cacheProvider.Set(ctx, key, "to-be-deleted", 10*time.Second)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	err = cacheProvider.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	exists, err := cacheProvider.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Fatal("expected key to not exist after delete")
	}
}

func TestCacheProvider_Exists(t *testing.T) {
	ctx := context.Background()
	key := "test:exists"

	err := cacheProvider.Set(ctx, key, "present", 10*time.Second)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	exists, err := cacheProvider.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Fatal("expected key to exist")
	}

	// cleanup
	_ = cacheProvider.Delete(ctx, key)
}

func TestCacheProvider_GetNonExistent(t *testing.T) {
	ctx := context.Background()
	key := "test:non-existent-key-" + t.Name()

	val, err := cacheProvider.Get(ctx, key)
	if err == nil {
		t.Fatal("expected error for non-existent key, got nil")
	}
	if err != redis.Nil {
		t.Fatalf("expected redis.Nil error, got %v", err)
	}
	if val != "" {
		t.Fatalf("expected empty string for non-existent key, got %q", val)
	}
}

func TestCacheProvider_SetWithTTL(t *testing.T) {
	ctx := context.Background()
	key := "test:ttl-expiry"

	err := cacheProvider.Set(ctx, key, "expires-soon", 1*time.Second)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// verify key exists immediately
	exists, err := cacheProvider.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Fatal("expected key to exist immediately after set")
	}

	// wait for TTL to expire
	time.Sleep(2 * time.Second)

	exists, err = cacheProvider.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists after TTL failed: %v", err)
	}
	if exists {
		t.Fatal("expected key to have expired after TTL")
	}
}
