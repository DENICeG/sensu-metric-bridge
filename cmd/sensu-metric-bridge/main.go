package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/danielb42/whiteflag"
)

var (
	prmMeasurementName  string
	prmFromEndpoint     string
	prmRelevantPrefixes string

	nowTS = strconv.FormatInt(time.Now().UnixNano(), 10)
)

type TagSet struct {
	Key   string
	Value string
}

func main() {

	whiteflag.Alias("m", "measurementName", "name of InfluxDB measurement")
	whiteflag.Alias("f", "fromEndpoint", "endpoint to scrape (http://url:port/path)")
	whiteflag.Alias("r", "relevantPrefix", "which metrics to consider in endpoint output (comma-separated)")

	if !whiteflag.CheckString("m") || !whiteflag.CheckString("f") {
		println("usage: sensu-metric-bridge -m <measurementName> -f <fromEndpoint> [-r <relevantPrefix>]")
		os.Exit(1)
	}

	prmMeasurementName = whiteflag.GetString("m")
	prmFromEndpoint = whiteflag.GetString("f")

	if whiteflag.CheckString("r") {
		prmRelevantPrefixes = whiteflag.GetString("r")
	}

	resp, err := http.Get(prmFromEndpoint)
	if err != nil {
		log.Println("could not scrape metrics from", prmFromEndpoint, err.Error())
		os.Exit(2)
	}
	defer resp.Body.Close()

	var output string
	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		line := s.Text()
		if metricHasRelevantPrefix(line) {
			output += transformPrometheusToInflux(line) + "\n"
		}
	}

	fmt.Println(output)
}

func metricHasRelevantPrefix(metric string) bool {

	var hasRelevantPrefix bool

	for _, relevantPrefix := range strings.Split(prmRelevantPrefixes, ",") {
		relevantPrefix = strings.TrimSpace(relevantPrefix)

		if strings.HasPrefix(metric, relevantPrefix) {
			hasRelevantPrefix = true
			break
		}
	}

	return hasRelevantPrefix
}

func transformPrometheusToInflux(metric string) string {

	line := strings.Split(metric, " ")
	metricLHS := line[0]
	metricRHS := line[1]

	// Case: simple case, like
	//   seconds_since_last_successful_run 46598.538422381
	if !strings.Contains(metric, "{") {
		return prmMeasurementName + ",item=" + metricLHS + " value=" + metricRHS + " " + nowTS
	}

	// Case: metric with fields and only the relevantPrefix as identifier, like
	//   metrics_DBPuller{domain="DB",item="TransactionsTotal"} 17
	var tags string
	for _, tag := range extractTags(metric) {
		tags += "," + tag.Key + "=" + tag.Value
	}

	return prmMeasurementName + tags + " value=" + metricRHS + " " + nowTS

	// Case: metric with fields, relevantPrefix and another constant identifier, like
	//    contactvalidator_return_proc{field="files",result="err"} 0
	// TODO
}

func extractTags(metric string) []TagSet {

	var tagSet []TagSet
	tagRE := regexp.MustCompile(`(\w+)="(\w+)"`)

	for _, matches := range tagRE.FindAllStringSubmatch(metric, -1) {
		t := TagSet{
			Key:   matches[1],
			Value: matches[2],
		}

		tagSet = append(tagSet, t)
	}

	return tagSet
}
