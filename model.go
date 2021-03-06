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
	Jenkins            string `json:"-"`
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
		log.Println(url, "Request failed:", err)
		items := make([]*job, 1)
		items[0] = &job{
			Name:    fmt.Sprintf("%cRequest failed: %s (%s)", 9, err, url),
			Jenkins: url,
		}
		jobs.Jobs = items
		return jobs
	}

	if resp.StatusCode != http.StatusOK {
		log.Println(url, "Reponse was not OK:", resp.StatusCode)
		items := make([]*job, 1)
		items[0] = &job{
			Name:    fmt.Sprintf("%cReponse was not OK: %d (%s)", 9, resp.StatusCode, url),
			Jenkins: url,
		}
		jobs.Jobs = items
		return jobs
	}

	time.AfterFunc(10*time.Second, func() {
		resp.Body.Close()
	})

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&jobs)
	if err != nil {
		log.Println(url, err)
		items := make([]*job, 1)
		items[0] = &job{
			Name:    fmt.Sprintf("%cResponse could not be decoded: %s (%s)", 9, err, url),
			Jenkins: url,
		}
		jobs.Jobs = items
	}

	var deleteIdx []int
	for i := 0; i < len(jobs.Jobs); i++ {
		jobs.Jobs[i].Jenkins = url
		if jobs.Jobs[i].FullName != "" {
			jobs.Jobs[i].Name = jobs.Jobs[i].FullName
		}
		if jobs.Jobs[i].Class == "com.cloudbees.hudson.plugins.folder.Folder" ||
			jobs.Jobs[i].Class == "jenkins.branch.OrganizationFolder" ||
			jobs.Jobs[i].Class == "org.jenkinsci.plugins.workflow.multibranch.WorkflowMultiBranchProject" {
			deleteIdx = append(deleteIdx, i)
		}
	}
	jobs.Jobs = deleteFromJobsArray(jobs.Jobs, deleteIdx...)

	return jobs
}

func deleteFromJobsArray(input []*job, indexes ...int) []*job {
	var output []*job
	lastIdx := 0
	for _, idx := range indexes {
		output = append(output, input[lastIdx:idx]...)
		lastIdx = idx + 1
	}
	output = append(output, input[lastIdx:]...)
	return output
}
