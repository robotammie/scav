package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/atomic"
)

//go:embed done.txt
var _alreadyDoneStr string

var _alreadyDone = make(map[string]struct{})

func init() {
	for _, w := range strings.Split(_alreadyDoneStr, "\n") {
		_alreadyDone[w] = struct{}{}
	}
}

func main() {
	log.SetFlags(0)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type solver struct {
	client *http.Client
}

func run() error {
	cookies, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("initialize cookie jar: %v", err)
	}

	u, err := url.Parse("http://buzz.pythonanywhere.com/word/")
	if err != nil {
		return err
	}

	cookies.SetCookies(u, []*http.Cookie{
		{
			Name:  "csrftoken",
			Value: "K1nE3fd60klUxlusXcwRiTp7Q53Pexr1CzAKI2of6pvIX6KwNhkTLgSfN7gsE2ZU",
		},
		{
			Name:  "sessionid",
			Value: "nfjvv8pziw0qi6zztf0xdl6r7lbu2ine",
		},
	})

	solv := solver{
		client: &http.Client{
			Jar: cookies,
			// Timeout: 5*time.Second,
			},
		}

		// for _, word := range _words {
		//     if err := solv.TryWord(word); err != nil {
		//         return err
		//     }
		// }

		const N = 8
		var (
			wg  sync.WaitGroup
			idx atomic.Int64
		)
		ctx := context.Background()
		wg.Add(N)
		for i := 0; i < N; i++ {
			go func() {
				defer wg.Done()

				for {
					idx := int(idx.Inc() - 1)
					if idx >= len(_words) {
						return
					}

					var err error
					for {
						err = solv.TryWord(ctx, _words[idx])
						if err == nil {
							break
						}
						if errors.Is(err, ErrRetry) {
							time.Sleep(500 * time.Millisecond)
							continue
					}
					log.Printf("worker dying: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

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
	pushWordsOfLength(nil, 7)
}

type payload struct {
	Word string `json:"numeric_word"`
}

var ErrRetry = errors.New("forbidden")

func (s *solver) TryWord(ctx context.Context, wordIndexes []int) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	var wordstr []byte
	for _, c := range wordIndexes {
		wordstr = strconv.AppendInt(wordstr, int64(c), 10)
	}
	word := string(wordstr)
	if _, done := _alreadyDone[word]; done {
		return nil
	}

	payload, err := json.Marshal(payload{Word: word})
	if err != nil {
		return fmt.Errorf("format JSON: %v", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"http://buzz.pythonanywhere.com/word/",
		bytes.NewReader(payload),
	)
	if err != nil {
		return err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	res, err := s.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("%s: timed out; will retry", wordstr)
			return ErrRetry
		}
		return fmt.Errorf("cannot send request: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusForbidden {
		log.Printf("%s: forbidden; will retry", wordstr)
		return ErrRetry
	}

	var response struct {
		Message string `json:"message"`
		Score   int    `json:"score"`
		Count   int    `json:"count"`
		Victory bool   `json:"victory"`
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return fmt.Errorf("decode response: %v", err)
	}

	if response.Message == "Good job!" {
		log.Printf("%s: GOOD JOB (SCORE: %d, COUNT: %d, VICTORY: %v)",
		wordstr, response.Score, response.Count, response.Victory)
	} else {
		log.Printf("%s: %s", wordstr, response.Message)
	}

	return nil
}