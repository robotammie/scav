package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"go.uber.org/atomic"
)

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	bs, err := loadBotSet("accounts.json")
	if err != nil {
		return err
	}
	defer bs.Close()

	co := newCoordinator(bs)
	defer co.Save()

	// Reporter state every second.
	go func() {
		for range time.Tick(time.Second) {
			log.Printf("status: good %d, bad %d", co.goodWords.Load(), co.badWords.Load())
		}
	}()

	const N = 8
	var (
		// wg   sync.WaitGroup
		idx  atomic.Int64
		quit atomic.Bool
	)
	ctx := context.Background()
	// wg.Add(N)
	for i := 0; i < N; i++ {
		go func(workerIdx int) {
			// defer wg.Done()

			for !quit.Load() {
				idx := int(idx.Inc() - 1)
				if idx >= len(_words) {
					return
				}

				var word []byte
				for _, c := range _words[idx] {
					word = strconv.AppendInt(word, int64(c), 10)
				}

				if err := co.TryWord(ctx, string(word)); err != nil {
					log.Printf("worker %d: %v", workerIdx, err)
					return
				}

				// time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	log.Printf("Got interrupted. Stopping.")
	// wg.Wait()

	return nil
}

var _words [][]int

func pushWordsOfLength(word []int, length int) {
	if len(word) == length {
		_words = append(_words, word)
		return
	}

	for i := 1; i <= 7; i++ {
		newWord := make([]int, 0, len(word)+1)
		newWord = append(newWord, word...)
		newWord = append(newWord, i)
		pushWordsOfLength(newWord, length)
	}
}

func init() {
	pushWordsOfLength(nil, 6)
}
