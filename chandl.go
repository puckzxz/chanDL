package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/gosuri/uiprogress"
)

// Thread contains the information and the download path of the thread
type Thread struct {
	URL          string
	Board        string
	ID           string
	Images       []string
	DownloadPath string
	FolderName   string
}

// ParseURL parses the raw URL and extracts data from it
func (t *Thread) ParseURL() {
	t.Board = strings.Split(strings.Split(t.URL, "org/")[1], "/")[0]
	t.ID = strings.Split(t.URL, "thread/")[1]
	if strings.Contains(t.ID, "/") {
		t.ID = strings.Split(t.ID, "/")[0]
	}
	t.FolderName = fmt.Sprintf("%s_%s", t.Board, t.ID)
	if !strings.HasSuffix(t.DownloadPath, "/") {
		t.DownloadPath = fmt.Sprintf("%s/", t.DownloadPath)
	}
	t.DownloadPath = filepath.FromSlash(fmt.Sprintf(`%s%s`, t.DownloadPath, t.FolderName))
}

// GetImages retrieves the URL's to all the images in the thread
func (t *Thread) GetImages() {
	c := colly.NewCollector(
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 2})
	c.OnHTML(".fileThumb", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		link = fmt.Sprintf("https:%s", link)
		t.Images = append(t.Images, link)
	})
	c.Visit(t.URL)
	c.Wait()
}

// Download downloads the thread
func (t *Thread) Download() {
	t.ParseURL()
	t.GetImages()
	if _, err := os.Stat(t.DownloadPath); os.IsNotExist(err) {
		err = os.Mkdir(t.DownloadPath, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}
	uiprogress.Start()
	bar := uiprogress.AddBar(len(t.Images)).PrependElapsed()
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%d - %d", b.Current(), len(t.Images))
	})
	var wg sync.WaitGroup
	wg.Add(len(t.Images))
	for _, image := range t.Images {
		fileName := filepath.FromSlash(fmt.Sprintf(`%s/%s`, t.DownloadPath, path.Base(image)))
		go func(url string, filename string, wg *sync.WaitGroup, pgbar *uiprogress.Bar) error {
			// Create the file on the disk
			file, err := os.Create(filename)
			if err != nil {
				return err
			}
			defer file.Close()
			// Get the data from our URL
			resp, err := http.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			// Write the data to our file
			_, err = io.Copy(file, resp.Body)
			if err != nil {
				return err
			}
			defer wg.Done()
			pgbar.Incr()
			return nil
		}(image, fileName, &wg, bar)
	}
	wg.Wait()
	uiprogress.Stop()
	fmt.Printf("Downloaded %d images to %s\n", len(t.Images), t.DownloadPath)
}

func main() {
	t := &Thread{}
	currentDir, _ := os.Getwd()
	thread := flag.String("thread", "", "Thread to download from. (Required)")
	dlPath := flag.String("path", currentDir, "Where to put the folder containing the images. Defaults to current directory")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()
	if *thread == "" {
		log.Fatal("[!] You must supply a 4chan thread")
	}
	t.URL = *thread
	t.DownloadPath = *dlPath
	t.Download()
}
