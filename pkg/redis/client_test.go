package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

// TestClientOperations verifies basic Redis storage operations.
func TestClientOperations(t *testing.T) {
	server := miniredis.RunT(t)
	client := New(Config{Address: server.Addr()})
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %v", err)
		}
	}()

	ctx := context.Background()
	key := "pixels:test"
	value := []byte("payload")

	if err := client.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("set value: %v", err)
	}

	found, ok, err := client.Find(ctx, key)
	if err != nil {
		t.Fatalf("find value: %v", err)
	}

	if !ok {
		t.Fatal("expected value to exist")
	}

	if string(found) != string(value) {
		t.Fatalf("expected value %q, got %q", value, found)
	}

	if err := client.Expire(ctx, key, time.Second); err != nil {
		t.Fatalf("expire value: %v", err)
	}

	if err := client.Delete(ctx, key); err != nil {
		t.Fatalf("delete value: %v", err)
	}

	_, ok, err = client.Find(ctx, key)
	if err != nil {
		t.Fatalf("find missing value: %v", err)
	}

	if ok {
		t.Fatal("expected deleted value to be missing")
	}
}

// TestClientTake verifies atomic read and delete.
func TestClientTake(t *testing.T) {
	server := miniredis.RunT(t)
	client := New(Config{Address: server.Addr()})
	defer func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %v", err)
		}
	}()

	ctx := context.Background()
	key := "pixels:take"
	value := []byte("payload")

	if err := client.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("set value: %v", err)
	}

	found, ok, err := client.Take(ctx, key)
	if err != nil {
		t.Fatalf("take value: %v", err)
	}

	if !ok || string(found) != string(value) {
		t.Fatalf("expected taken value %q, got %q", value, found)
	}

	_, ok, err = client.Take(ctx, key)
	if err != nil {
		t.Fatalf("take missing value: %v", err)
	}

	if ok {
		t.Fatal("expected taken value to be missing")
	}
}

// TestClientIncrementPreservesFirstExpiration verifies atomic counters keep their first window.
func TestClientIncrementPreservesFirstExpiration(t *testing.T) {
	server := miniredis.RunT(t)
	client := New(Config{Address: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	for expected := int64(1); expected <= 3; expected++ {
		value, err := client.Increment(ctx, "pixels:counter", time.Minute)
		if err != nil || value != expected {
			t.Fatalf("increment expected=%d value=%d err=%v", expected, value, err)
		}
	}
	if ttl := server.TTL("pixels:counter"); ttl != time.Minute {
		t.Fatalf("expected stable ttl, got %s", ttl)
	}
	created, err := client.SetIfAbsent(ctx, "pixels:lock", []byte{'1'}, time.Minute)
	if err != nil || !created {
		t.Fatalf("set lock created=%v err=%v", created, err)
	}
	created, err = client.SetIfAbsent(ctx, "pixels:lock", []byte{'2'}, time.Hour)
	stored, getErr := server.Get("pixels:lock")
	if err != nil || getErr != nil || created || stored != "1" {
		t.Fatalf("expected existing lock preserved created=%v err=%v", created, err)
	}
}

// TestClientListAndSetOperations verifies ordered lists and unique set members.
func TestClientListAndSetOperations(t *testing.T) {
	server := miniredis.RunT(t)
	client := New(Config{Address: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()

	if err := client.ListAppend(ctx, "pixels:list", []byte("one"), []byte("two")); err != nil {
		t.Fatalf("append list: %v", err)
	}
	length, err := client.ListLength(ctx, "pixels:list")
	if err != nil || length != 2 {
		t.Fatalf("length=%d err=%v", length, err)
	}
	values, err := client.ListRange(ctx, "pixels:list", 0, -1)
	if err != nil || len(values) != 2 || string(values[0]) != "one" || string(values[1]) != "two" {
		t.Fatalf("values=%q err=%v", values, err)
	}

	if err = client.SetAdd(ctx, "pixels:set", "one", "two", "one"); err != nil {
		t.Fatalf("add set: %v", err)
	}
	members, err := client.SetMembers(ctx, "pixels:set")
	if err != nil || len(members) != 2 {
		t.Fatalf("members=%q err=%v", members, err)
	}
	if err = client.SetRemove(ctx, "pixels:set", "one"); err != nil {
		t.Fatalf("remove set: %v", err)
	}
	members, err = client.SetMembers(ctx, "pixels:set")
	if err != nil || len(members) != 1 || members[0] != "two" {
		t.Fatalf("remaining=%q err=%v", members, err)
	}
}
