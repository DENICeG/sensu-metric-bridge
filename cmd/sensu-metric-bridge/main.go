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

	if !whiteflag.FlagPresent("m") || !whiteflag.FlagPresent("f") {
		println("usage: sensu-metric-bridge -m <measurementName> -f <fromEndpoint> [-r <relevantPrefix>]")
		os.Exit(1)
	}

	prmMeasurementName = whiteflag.GetString("m")
	prmFromEndpoint = whiteflag.GetString("f")
	prmRelevantPrefixes = whiteflag.GetString("r")

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

		for _, relevantPrefix := range strings.Split(prmRelevantPrefixes, ",") {
			if metricEqualsRelevantPrefix(line, relevantPrefix) {
				output += transformPrometheusToInfluxCaseA(line) + "\n"
			} else if metricHasRelevantPrefixOnly(line, relevantPrefix) {
				output += transformPrometheusToInfluxCaseB(line) + "\n"
			} else if metricHasRelevantPrefixAndAdditionalIdentifier(line, relevantPrefix) {
				output += transformPrometheusToInfluxCaseC(line, relevantPrefix) + "\n"
			}
		}
	}

	fmt.Println(output)
}

func metricEqualsRelevantPrefix(metric, relevantPrefix string) bool {
	return strings.HasPrefix(metric, relevantPrefix+" ")
}

func metricHasRelevantPrefixOnly(metric, relevantPrefix string) bool {
	return strings.HasPrefix(metric, relevantPrefix+"{")
}

func metricHasRelevantPrefixAndAdditionalIdentifier(metric, relevantPrefix string) bool {
	return strings.HasPrefix(metric, relevantPrefix+"_")
}

// Case: simple case, like
//   seconds_since_last_successful_run 46598.538422381
func transformPrometheusToInfluxCaseA(metric string) string {

	line := strings.Split(metric, " ")
	metricLHS := line[0]
	metricRHS := line[1]

	return prmMeasurementName + ",item=" + metricLHS + " value=" + metricRHS + " " + nowTS
}

// Case: metric with fields and only the relevantPrefix as identifier, like
//   metrics_DBPuller{domain="DB",item="TransactionsTotal"} 17
func transformPrometheusToInfluxCaseB(metric string) string {

	var tags string
	for _, tag := range extractTags(metric) {
		tags += "," + tag.Key + "=" + tag.Value
	}

	metricValue := strings.Split(metric, " ")[1]
	return prmMeasurementName + tags + " value=" + metricValue + " " + nowTS
}

// Case: metric with fields, relevantPrefix and another constant identifier, like
//    contactvalidator_return_proc{field="files",result="err"} 0
func transformPrometheusToInfluxCaseC(metric, relevantPrefix string) string {

	var tags string
	for _, tag := range extractTags(metric) {
		tags += "," + tag.Key + "=" + tag.Value
	}

	metricValue := strings.Split(metric, " ")[1]
	additionalIdentifier := strings.TrimPrefix(metric, relevantPrefix+"_")
	additionalIdentifier = strings.Split(additionalIdentifier, "{")[0]
	additionalIdentifier = strings.Split(additionalIdentifier, " ")[0]
	return prmMeasurementName + ",item=" + additionalIdentifier + tags + " value=" + metricValue + " " + nowTS
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
