package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"go.uber.org/atomic"
)

type coordinator struct {
	bs        *botSet
	mu        sync.Mutex
	matches   []string
	badWords  atomic.Int64
	goodWords atomic.Int64
}

func newCoordinator(bs *botSet) *coordinator {
	return &coordinator{bs: bs}
}

func (co *coordinator) Save() error {
	f, err := os.OpenFile("good.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, w := range co.matches {
		fmt.Fprintln(f, w)
	}
	return nil
}

// TryWord is a thread-safe version of TryWord that sends multiple bot
// requests.
func (c *coordinator) TryWord(ctx context.Context, word string) error {
	b := c.bs.NextBot()
	attempts := 0
	for {
		err := b.TryWord(ctx, word)
		if err == nil {
			c.mu.Lock()
			c.matches = append(c.matches, word)
			c.goodWords.Inc()
			c.mu.Unlock()
			return nil
		}

		if errors.Is(err, ErrBadWord) {
			c.badWords.Inc()
			return nil
		}

		if errors.Is(err, ErrRateLimited) {
			b = c.bs.NextBot()
			attempts++
			if attempts < 5 {
				continue
			}

			log.Printf("too many failed attempts; creating new bot")
			b, err = c.bs.NewBot()
			if err == nil {
				continue
			}
		}

		return err
	}
}

type botSet struct {
	path string
	mu   sync.Mutex
	bots []*bot
	idx  int
}

func loadBotSet(path string) (*botSet, error) {
	bs := botSet{path: path}

	f, err := os.Open(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if err == nil {
		defer f.Close()

		var bots []struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(f).Decode(&bots); err != nil {
			return nil, fmt.Errorf("decode %q: %v", path, err)
		}

		for _, bdesc := range bots {
			b, err := newBot(bdesc.Name, bdesc.Email)
			if err != nil {
				return nil, fmt.Errorf("create bot %v/%v: %v", bdesc.Name, bdesc.Email, err)
			}
			bs.bots = append(bs.bots, b)
		}
	}

	// Seed with a single starting bot.
	if len(bs.bots) == 0 {
		log.Printf("there are no bots. creating one.")
		b, err := newBot("Bot 1", "notreal@example.com")
		if err != nil {
			return nil, fmt.Errorf("create bot: %v", err)
		}
		bs.bots = append(bs.bots, b)
	}

	return &bs, nil
}

func (bs *botSet) Close() error {
	f, err := os.Create(bs.path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	bots := make([]struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}, len(bs.bots))
	for i, b := range bs.bots {
		bots[i].Name = b.name
		bots[i].Email = b.email
	}

	return enc.Encode(bots)
}

func (bs *botSet) NextBot() *bot {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	idx := bs.idx
	bs.idx = (bs.idx + 1) % len(bs.bots)
	return bs.bots[idx]
}

func (bs *botSet) NewBot() (*bot, error) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	botIdx := len(bs.bots) + 1
	b, err := newBot(
		fmt.Sprintf("Bot %d", botIdx),
		fmt.Sprintf("gashbot%d@example.com", botIdx),
	)
	if err != nil {
		return nil, fmt.Errorf("create bot %d: %v", botIdx, err)
	}

	bs.bots = append(bs.bots, b)
	return b, nil
}
