package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

type jobs struct {
	//XMLName xml.Name `xml:"Projects"`
	Jobs []job `xml:"Project"`
}

type job struct {
	//XMLName   xml.Name  `xml:"Project"`
	checked   bool
	Name      string    `xml:"name,attr"`
	Status    string    `xml:"lastBuildStatus,attr"`
	Label     string    `xml:"lastBuildLabel,attr"`
	Activity  string    `xml:"activity,attr"`
	Url       string    `xml:"webUrl,attr"`
	BuildTime time.Time `xml:"lastBuildTime,attr"`
}

func getCCXmlJobs(url string) jobs {
	resp, err := http.Get(url)
	jobs := jobs{Jobs: []job{}}
	if err != nil {
		log.Println(err)
		items := make([]job, 1)
		items[0] = job{Name: err.Error()}
		jobs.Jobs = items
		return jobs
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Antwort war nicht OK")
		items := make([]job, 1)
		items[0] = job{Name: fmt.Sprintf("Antwort war nicht OK: %d", resp.StatusCode)}
		jobs.Jobs = items
		return jobs
	}
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(&jobs)
	if err != nil {
		log.Println(err)
		items := make([]job, 1)
		items[0] = job{Name: "Antwort konnte nicht decodiert werden: " + err.Error()}
		jobs.Jobs = items
	}
	return jobs
}

func getLastUnstable(newJob *job) (string, error) {
	resp, err := http.Get(newJob.Url + "api/xml?xpath=/*/lastUnstableBuild/number")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New(fmt.Sprintf("Antwort nicht OK: %d, %s", resp.StatusCode, newJob.Url+"api/xml?xpath=/*/lastUnstableBuild/number"))
	}
	decoder := xml.NewDecoder(resp.Body)
	var number string
	err = decoder.Decode(&number)
	if err != nil {
		return "", err
	}
	return number, nil
}
