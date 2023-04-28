package main

import (
	"log"
	"os"
	"path/filepath"
	"encoding/json"
	"fmt"
	"net/http"
	"io"
	"sync"
)

const WORKERCOUNT = 4

func usage() {
	fmt.Fprintln(os.Stderr, "usage: sifter [JSON file from Wireshark] [extraction target directory]")
	os.Exit(1)
}

func Decode(jsonFile *os.File) { // Decode JSON file and send jobs to DownloadWorker
	var wg sync.WaitGroup

	type HTTP struct {
		URI string `json:"http.request.full_uri"`
	}

	type Frame struct {
		Time string `json:"frame.time"`
	}

	type IP struct{
		Addr string `json:"ip.addr"`
	}

	type Layer struct {
		HTTP HTTP `json:"http"`
		Frame Frame `json:"frame"`
		IP IP `json:"ip"`
	}

	type Source struct {
		Layer Layer `json:"layers"`
	}

	type Asset []struct {
		Source Source `json:"_source"`
	}

	decoder := json.NewDecoder(jsonFile)
	for {
		var asset Asset
		if err := decoder.Decode(&asset); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}


		jobs := make(chan string)
		// Spawn workers
		for i := 0; i < WORKERCOUNT; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				DownloadWorker(jobs)
			}()
		}

		for i := 0; i < len(asset); i++ {
			log.Println("Worker downloading asset:",
				"\nRequest TIME: " , asset[i].Source.Layer.Frame.Time,
				"\nRequest IP: ", asset[i].Source.Layer.IP.Addr,
				"\nRequest URI: ", asset[i].Source.Layer.HTTP.URI,
				"\n",
			)
			jobs <- (asset[i].Source.Layer.HTTP.URI)
		}
		close(jobs)
	}
	wg.Wait()
}

// Graciously stolen from vinegar
func DownloadWorker(request <-chan string) error {
	for job := range request {


		targetFile := filepath.Join(os.Args[2], job[22:])

		out, err := os.Create(targetFile)
		if err != nil {
			return err
		}

		resp, err := http.Get(job)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("bad status: %s", resp.Status)
		}

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if len(os.Args) < 3 {
		usage()
	}	
	
	json, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer json.Close()
	
	// Simple check if target dir exists
	_, err = os.Stat(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	Decode(json)
}
