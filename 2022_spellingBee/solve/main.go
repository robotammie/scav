package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

//go:embed matches.txt
var _matchData string

var _matches []string

func init() {
	for _, m := range strings.Split(_matchData, "\n") {
		m = strings.TrimSpace(m)
		if len(m) == 0 {
			continue
		}
		_matches = append(_matches, m)
	}
}

func main() {
	log.SetFlags(0)
	if err := new(mainCmd).Run(); err != nil {
		log.Fatal(err)
	}
}

type mainCmd struct {
	client    *http.Client
	csrfToken string
}

func (cmd *mainCmd) init() error {
	cookies, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("initialize cookie jar: %v", err)
	}

	cmd.client = &http.Client{Jar: cookies}

	res, err := cmd.client.Get("http://buzz.pythonanywhere.com/")
	if err != nil {
		return fmt.Errorf("initialize: %v", err)
	}
	defer res.Body.Close()

	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		return err
	}

	u, err := url.Parse("http://buzz.pythonanywhere.com/")
	if err != nil {
		log.Panicf("impossible: %v", err)
	}

	for _, c := range cookies.Cookies(u) {
		log.Printf("\t%v: %v", c.Name, c.Value)
		if c.Name == "csrftoken" {
			cmd.csrfToken = c.Value
		}
	}

	if len(cmd.csrfToken) == 0 {
		return errors.New("did not receive a csrf token")
	}

	return nil
}

func (cmd *mainCmd) Run() error {
	if err := cmd.init(); err != nil {
		return err
	}

	if err := cmd.login(); err != nil {
		return fmt.Errorf("cannot login: %v", err)
	}

	ctx := context.Background()
outer:
	for _, word := range _matches {
		if len(word) < 4 {
			continue
		}
		for {
			err := cmd.Submit(ctx, word)
			if err == nil {
				time.Sleep(100 * time.Millisecond)
				continue outer
			}

			if errors.Is(err, ErrRateLimited) {
				log.Printf("rate limited. retrying")
				time.Sleep(time.Second)
			} else {
				return fmt.Errorf("word %v failed: %v", word, err)
			}
		}
	}

	return nil
}

func (cmd *mainCmd) login() error {
	log.Printf("logging in")
	res, err := cmd.client.PostForm("http://buzz.pythonanywhere.com/login/?next=/", url.Values{
		"csrfmiddlewaretoken": {cmd.csrfToken},
		"form-type":           {"login"},
		"email":               {"rowan@blinkers-off.com"},
	})
	if err != nil {
		return fmt.Errorf("send request: %v", err)
	}
	defer res.Body.Close()

	var buff bytes.Buffer
	if _, err := io.Copy(&buff, res.Body); err != nil {
		return fmt.Errorf("read response: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("http %v: %v\nresponse: %s", res.StatusCode, res.Status, buff.String())
	}

	return nil
}

var ErrRateLimited = errors.New("rate limited")

func (cmd *mainCmd) Submit(ctx context.Context, word string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	var request struct {
		Word string `json:"numeric_word"`
	}
	request.Word = word

	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("serialize JSON: %v", err)
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

	res, err := cmd.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrRateLimited
		}
		return fmt.Errorf("word %q failed: %w", request.Word, err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusForbidden {
		return ErrRateLimited
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

	switch response.Message {
	case "You already found this word!":
		log.Printf("%s: already found", word)
		return nil

	case "Good job!":
		log.Printf("%s: SCORE: %d, COUNT: %d, VICTORY: %v",
			request.Word, response.Score, response.Count, response.Victory)
		return nil
	}

	return errors.New("bad word")
}
