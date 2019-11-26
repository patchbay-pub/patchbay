package patchbay

import (
        "fmt"
        "log"
        "io/ioutil"
        "strings"
        "net/http"
        "os"
        "path"
)

const NumWorkers = 4

var ValidExt = [...]string{".html", ".js", ".ico", ".css", ".jpg", ".svg"}

type Hoster struct {
        rootChannel string
        dir string
        client *http.Client
        authToken string
}

func (h *Hoster) Start() {
        h.HostDir(h.rootChannel, h.dir, NumWorkers)
}

func (h *Hoster) HostDir(channel string, dirPath string, numWorkers int) {

        entries, err := ioutil.ReadDir(dirPath)
        if err != nil {
                log.Fatal(err)
        }

        for _, entry := range entries {
                if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
                        h.HostDir(channel + "/" + entry.Name(), path.Join(dirPath, entry.Name()), NumWorkers)
                } else {
                        if validExt(entry.Name()) {
                                h.HostFile(channel + "/" + entry.Name(), path.Join(dirPath, entry.Name()), NumWorkers)
                        }
                        // also host index files directly on the path
                        if entry.Name() == "index.html" {
                                //h.HostFile(channel, path.Join(dirPath, entry.Name()), NumWorkers)
                                h.HostFile(channel + "/", path.Join(dirPath, entry.Name()), NumWorkers)
                        }
                }
        }
}

func (h *Hoster) HostFile(channel string, path string, numWorkers int) {

        for i := 0; i < numWorkers; i++ {
                go func(index int) {
                        for {
                                file, err := os.Open(path)
                                if err != nil {
                                        log.Fatal(err)
                                }

                                req, err := http.NewRequest("POST", channel, file)
                                if err != nil {
                                        log.Fatal(err)
                                }

                                if h.authToken != "" {
                                        req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.authToken))
                                }

                                res, err := h.client.Do(req)
                                if err != nil {
                                        log.Fatal(err)
                                }

                                log.Println(fmt.Sprintf("Served %s on channel %s from worker %d", path, channel, index))

                                if res.StatusCode > 299 {
                                        log.Println("Something went wrong")
                                }

                        }
                }(i)
        }
}

func validExt(path string) bool {
        for _, ext := range ValidExt {
                if strings.HasSuffix(path, ext) {
                        return true
                }
        }
        return false
}




type HosterBuilder struct {
        hoster Hoster
}

func (h *HosterBuilder) Dir(dir string) *HosterBuilder {
        h.hoster.dir = dir
        return h
}

func (h *HosterBuilder) RootChannel(channel string) *HosterBuilder {
        h.hoster.rootChannel = channel
        return h
}

func (h *HosterBuilder) AuthToken(token string) *HosterBuilder {
        h.hoster.authToken = token
        return h
}

func (h *HosterBuilder) Build() *Hoster {
        return &h.hoster
}

func NewHosterBuilder() *HosterBuilder {
        return &HosterBuilder{hoster: Hoster{dir:".", client: &http.Client{}}}
}
