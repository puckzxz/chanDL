package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/gosuri/uiprogress"
)

func getImageURLs(threadURL string) []string {
	var links []string
	c := colly.NewCollector(
		colly.Async(true),
	)
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 2})
	c.OnHTML(".fileThumb", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		link = fmt.Sprintf("https:%s", link)
		links = append(links, link)
	})
	c.Visit(threadURL)
	c.Wait()
	return links
}

func downloadFile(filepath string, url string, wg *sync.WaitGroup, pgbar *uiprogress.Bar) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	defer wg.Done()

	pgbar.Incr()

	return nil
}

func main() {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	thread := flag.String("thread", "", "Thread to download from. (Required)")
	argPath := flag.String("path", currentDir, "Where to download files. Defaults to current directory")
	flag.Parse()

	if *thread == "" || !strings.Contains(*thread, "4chan") {
		log.Fatal("[!] You must supply a 4chan thread")
		os.Exit(1)
	}

	images := getImageURLs(*thread)

	board := strings.Split(strings.Split(*thread, "org/")[1], "/")[0]
	threadID := strings.Split(*thread, "thread/")[1]
	if strings.Contains(threadID, "/") {
		threadID = strings.Split(threadID, "/")[0]
	}

	dlFolder := fmt.Sprintf("%s - %s", board, threadID)

	dlPath := fmt.Sprintf(`%s\%s`, *argPath, dlFolder)

	if _, err := os.Stat(dlPath); os.IsNotExist(err) {
		err = os.Mkdir(dlPath, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	uiprogress.Start()
	bar := uiprogress.AddBar(len(images)).AppendCompleted()
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%d/%d", b.Current(), len(images))
	})
	var wg sync.WaitGroup
	wg.Add(len(images))
	for _, url := range images {
		filename := fmt.Sprintf(`%s`, path.Base(url))
		filePath := fmt.Sprintf(`%s\%s`, dlPath, filename)
		go downloadFile(filePath, url, &wg, bar)
	}
	wg.Wait()
	uiprogress.Stop()
	fmt.Printf("Downloaded %d images to %s\n", len(images), dlPath)
}
