package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/golog"
	"github.com/vharitonsky/iniflags"
)

const (
	etagFile = "http-proxy.etag"
)

var (
	autoUpdateURL = flag.String("updateurl", "", "URL at which to check for automatic updates")
	binaries      = flag.String("binaries", "binaries", "path in which to look for binaries, defaults to 'binaries'")
	goodAfter     = flag.Duration("goodafter", 5*time.Minute, "consider a given binary good after this amount of time (defaults to 5 minutes)")
	tries         = flag.Int("tries", 3, "How many times to retry with a given binary before deleting it")
	wait          = flag.Duration("wait", 5*time.Second, "amount of time to wait between successive invocations of http-proxy binary")
	help          = flag.Bool("help", false, "Get usage help")
)

var (
	log = golog.LoggerFor("http-proxy.supervisor")

	respawn = make(chan bool)
	wg      sync.WaitGroup
)

func main() {
	iniflags.SetAllowUnknownFlags(true)
	iniflags.Parse()
	if *help {
		flag.Usage()
		return
	}

	wg.Add(1)
	go keepRunning()
	if *autoUpdateURL != "" {
		go autoUpdate()
	}
	respawn <- true
	wg.Wait()
}

func keepRunning() {
	defer wg.Done()
	cmd := &exec.Cmd{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	for range respawn {
		try := 0
		for {
			infos, err := ioutil.ReadDir(*binaries)
			if err != nil {
				log.Fatalf("Unable to list binaries in %v: %v", *binaries, err)
			}

			if len(infos) == 0 {
				log.Fatalf("No binaries found in %v", *binaries)
			}
			path := filepath.Join(*binaries, infos[len(infos)-1].Name())
			cmd.Path = path
			start := time.Now()
			try++
			log.Debugf("Running binary at %v, attempt %d", path, try)
			runErr := cmd.Run()
			if runErr != nil {
				log.Errorf("Error running binary at %v: %v", path, runErr)
				delta := time.Now().Sub(start)
				if delta > *goodAfter {
					try = 0
				} else if try >= *tries {
					log.Debugf("Considering binary bad, removing: %v", path)
					rmErr := os.Remove(path)
					if rmErr != nil {
						log.Errorf("Error removing binary at %v: %v", path, rmErr)
					}
					try = 0
				}
			}
			time.Sleep(*wait)
		}
	}
}

func autoUpdate() {
	client := &http.Client{}
	etag := loadETag()
	for {
		time.Sleep(30 * time.Second)
		etag = fetchAutoUpdate(client, etag)
	}
}

func fetchAutoUpdate(client *http.Client, etag string) (newETag string) {
	newETag = etag
	url := *autoUpdateURL

	get, _ := http.NewRequest(http.MethodGet, url, nil)
	get.Header.Set("ETag", etag)
	resp, err := client.Do(get)
	if err != nil {
		log.Errorf("Unable to fetch updated version from %v: %v", url, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		log.Debug("Already on latest version")
		return
	}
	newETag = resp.Header.Get("ETag")
	newVersion := resp.Header.Get("X-Version")
	if newVersion == "" {
		log.Error("Response didn't include a version!")
		saveEtag(newETag)
		return
	}

	name, _ := getBinaryNameAndVersion()
	newFilename := fmt.Sprintf("%v.%v", name, newVersion)
	newBinary, err := os.OpenFile(newFilename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorf("Unable to create new binary at %v: %v", newFilename, err)
		return
	}
	_, err = io.Copy(newBinary, resp.Body)
	if err != nil {
		log.Errorf("Unable to save contents to new binary at %v: %v", newFilename, err)
		return
	}
	err = newBinary.Close()
	if err != nil {
		log.Errorf("Unable to finish saving contents to new binary at %v: %v", newFilename, err)
		return
	}
	saveEtag(etag)
	return
}

func loadETag() string {
	b, err := ioutil.ReadFile(etagFile)
	if err != nil {
		log.Debugf("Unable to read etag: %v", err)
		return ""
	}
	return string(b)
}

func saveEtag(etag string) {
	err := ioutil.WriteFile(etagFile, []byte(etag), 0644)
	if err != nil {
		log.Errorf("Unable to save etag: %v", err)
	}
}

func getBinaryPath() string {
	return os.Args[0]
}

func getBinaryNameAndVersion() (string, string) {
	path := getBinaryPath()
	dot := strings.LastIndex(path, ".")
	if dot < 0 {
		return path, ""
	}
	return path[:dot], path[dot+1:]
}
