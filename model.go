package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type jobs struct {
	Jobs []*job `json:"jobs"`
}

type job struct {
	Name               string `json:"displayName"`
	FullName           string `json:"fullDisplayName"`
	Color              string `json:"color"`
	URL                string `json:"url"`
	LastBuild          build  `json:"lastBuild,omitempty"`
	LastCompletedBuild build  `json:"lastCompletedBuild,omitempty"`
	Class              string `json:"_class,omitempty"`
}

type build struct {
	Building  bool      `json:"building"`
	Label     int       `json:"number"`
	Result    string    `json:"result,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

func (b *build) UnmarshalJSON(s []byte) error {
	type Alias build
	alias := struct {
		*Alias
		Timestamp int64 `json:"timestamp"`
	}{}
	err := json.Unmarshal(s, &alias)
	if err != nil {
		return err
	}
	if alias.Alias != nil {
		b.Building = alias.Building
		b.Label = alias.Label
		b.Result = alias.Result
		b.Timestamp = time.Unix(alias.Timestamp/1000, 0)
	}
	return nil
}

func getJobsFromMultiple(urls []string) jobs {
	var jobs jobs
	for _, url := range urls {
		j := getJobs(url)
		jobs.Jobs = append(jobs.Jobs, j.Jobs...)
	}
	return jobs
}

func getJobs(url string) jobs {
	resp, err := http.Get(url + "/api/json?tree=jobs[_class,fullDisplayName,displayName,url,color," +
		"lastBuild[number,timestamp,result,building],lastCompletedBuild[number,timestamp,result,building]]")
	var jobs jobs
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Println(err)
		items := make([]*job, 1)
		items[0] = &job{Name: err.Error()}
		jobs.Jobs = items
		return jobs
	}

	if resp.StatusCode != http.StatusOK {
		log.Println(url, "Antwort war nicht OK")
		items := make([]*job, 1)
		items[0] = &job{Name: fmt.Sprintf("%c%s: Antwort war nicht OK: %d", 9, url, resp.StatusCode)}
		jobs.Jobs = items
		return jobs
	}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&jobs)
	if err != nil {
		log.Println(url, err)
		items := make([]*job, 1)
		items[0] = &job{Name: fmt.Sprintf("%c%s: Antwort konnte nicht decodiert werden: %s", 9, url, err)}
		jobs.Jobs = items
	}

	var deleteIdx []int
	for i := 0; i < len(jobs.Jobs); i++ {
		if jobs.Jobs[i].Class == "com.cloudbees.hudson.plugins.folder.Folder" {
			deleteIdx = append(deleteIdx, i)
		}
	}
	jobs.Jobs = deleteFromJobsArray(jobs.Jobs, deleteIdx...)

	return jobs
}
