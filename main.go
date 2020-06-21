package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gosuri/uiprogress"
)

// Post contains the filename and extension of a post
type Post struct {
	Name       int    `json:"tim,omitempty"`
	Extenstion string `json:"ext"`
}

// Thread contains all the posts in a thread
type Thread struct {
	ID           string
	Board        string
	Posts        []*Post `json:"posts"`
	downloadPath string
}

// Filters a thread for all posts containing an image
func (t *Thread) filter() {
	temp := []*Post{}
	for _, p := range t.Posts {
		if p.Name != 0 {
			temp = append(temp, p)
		}
	}
	t.Posts = temp
}

func (t *Thread) parseURL(url string) {
	t.Board = strings.Split(strings.Split(url, "org/")[1], "/")[0]
	t.ID = strings.Split(url, "thread/")[1]

	// Sometimes the URL will contain the thread title
	// If it does, we remove it
	if strings.Contains(t.ID, "/") {
		t.ID = strings.Split(t.ID, "/")[0]
	}

	folder := fmt.Sprintf("%s_%s", t.Board, t.ID)

	// Add a trailing / because later we just append the filename to this
	if !strings.HasSuffix(t.downloadPath, "/") {
		t.downloadPath = fmt.Sprintf("%s/", t.downloadPath)
	}

	t.downloadPath = filepath.FromSlash(fmt.Sprintf(`%s%s`, t.downloadPath, folder))
}

// File returns a formatted filename, Ex. "123.png"
func (p *Post) File() string {
	return fmt.Sprintf("%d%s", p.Name, p.Extenstion)
}

func downloadFile(url string, filename string, wg *sync.WaitGroup, pb *uiprogress.Bar) error {
	// Create the file on the disk
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer file.Close()

	// Get the data from our URL
	resp, err := http.Get(url)

	// Sometimes 4chan will rate limit us
	// If we do get rate limited we just sleep for a bit
	for resp.StatusCode != 200 {
		time.Sleep(time.Millisecond * 250)
		resp, err = http.Get(url)
	}
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// Write the data to our file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	pb.Incr()

	defer wg.Done()

	return nil
}

// Download downloads all the images in a thread to disk
func (t *Thread) Download() error {
	t.filter()

	if _, err := os.Stat(t.downloadPath); os.IsNotExist(err) {
		err = os.Mkdir(t.downloadPath, 0777)
		if err != nil {
			log.Fatal(err)
			return err
		}
	}

	uiprogress.Start()
	bar := uiprogress.AddBar(len(t.Posts)).PrependElapsed()
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%d - %d", b.Current(), len(t.Posts))
	})

	var wg sync.WaitGroup
	wg.Add(len(t.Posts))

	for _, p := range t.Posts {
		// The full filepath including the filename, my/path/filename.png
		filePath := filepath.FromSlash(fmt.Sprintf(`%s/%s`, t.downloadPath, p.File()))
		// The URL on 4chan's servers of post's file
		url := fmt.Sprintf("https://i.4cdn.org/%s/%s", t.Board, p.File())
		go downloadFile(url, filePath, &wg, bar)
	}

	wg.Wait()
	uiprogress.Stop()

	return nil
}

func main() {
	currentDir, _ := os.Getwd()

	thread := flag.String("thread", "", "Thread to download from. (Required)")
	dlPath := flag.String("path", currentDir, "Where to put the folder containing the images. Defaults to current directory")

	flag.Usage = func() {
		flag.PrintDefaults()
	}

	flag.Parse()

	if *thread == "" {
		flag.PrintDefaults()
		return
	}

	t := &Thread{}

	t.downloadPath = *dlPath

	t.parseURL(*thread)

	resp, err := http.Get(fmt.Sprintf("https://a.4cdn.org/%s/thread/%s.json", t.Board, t.ID))
	if err != nil {
		panic(err)
	}

	json.NewDecoder(resp.Body).Decode(&t)

	t.Download()
}
