package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sync"
	"time"
)

type bot struct {
	name      string
	email     string
	cookies   http.CookieJar
	client    *http.Client
	csrfToken string
	log       *log.Logger

	mu   sync.Mutex
	once sync.Once
}

func newBot(name, email string) (*bot, error) {
	cookies, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("initialize cookie jar: %v", err)
	}

	b := &bot{
		name:    name,
		email:   email,
		cookies: cookies,
		client: &http.Client{
			Jar: cookies,
		},
		log: log.New(os.Stderr, "["+name+"] ", 0),
	}
	return b, nil
}

func (b *bot) init() (err error) {
	b.once.Do(func() {
		err = b.doInit()
	})
	return err
}

func (b *bot) doInit() error {
	if err := b.start(); err != nil {
		return fmt.Errorf("initialize bot %q: %v", b.name, err)
	}

	// create account or login
	if err := b.register(); err != nil {
		if errors.Is(err, errAccountExists) {
			err = b.login()
		}
		if err != nil {
			return fmt.Errorf("register/login bot: %v", err)
		}
	}

	return nil
}

func (b *bot) start() error {
	res, err := b.client.Get("http://buzz.pythonanywhere.com/")
	if err != nil {
		return fmt.Errorf("initialize: %v", err)
	}
	defer res.Body.Close()

	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		return err
	}

	u, err := url.Parse("http://buzz.pythonanywhere.com/")
	if err != nil {
		b.log.Panicf("impossible: %v", err)
	}

	cookies := b.cookies.Cookies(u)
	if len(cookies) == 0 {
		return errors.New("no cookies received on init")
	}

	b.log.Printf("cookies:")
	for _, c := range cookies {
		b.log.Printf("\t%v: %v", c.Name, c.Value)
		if c.Name == "csrftoken" {
			b.csrfToken = c.Value
		}
	}

	if len(b.csrfToken) == 0 {
		return errors.New("did not receive a csrf token")
	}

	return nil
}

var errAccountExists = errors.New("account already exists")

func (b *bot) register() error {
	res, err := b.client.PostForm("http://buzz.pythonanywhere.com/login/", url.Values{
		"csrfmiddlewaretoken": {b.csrfToken},
		"form-type":           {"registration"},
		"team_name":           {"GASH"},
		"your_name":           {b.name},
		"email":               {b.email},
	})
	if err != nil {
		return fmt.Errorf("send request: %v", err)
	}
	defer res.Body.Close()

	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		return fmt.Errorf("read response: %v", err)
	}

	if res.StatusCode == http.StatusInternalServerError {
		return errAccountExists
	}

	return nil
}

func (b *bot) login() error {
	res, err := b.client.PostForm("http://buzz.pythonanywhere.com/login/?next=/", url.Values{
		"csrfmiddlewaretoken": {b.csrfToken},
		"form-type":           {"login"},
		"email":               {b.email},
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

var (
	ErrRateLimited = errors.New("getting rate limited")
	ErrBadWord     = errors.New("bad word")
)

func (b *bot) TryWord(ctx context.Context, word string) error {
	if err := b.init(); err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

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

	res, err := b.client.Do(req)
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

	if response.Message == "Good job!" {
		b.log.Printf("%s: SCORE: %d, COUNT: %d, VICTORY: %v",
			request.Word, response.Score, response.Count, response.Victory)
		return nil
	}

	return ErrBadWord
}
