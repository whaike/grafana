package cloudwatch

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana/pkg/components/simplejson"
)

// Parses the json queries and returns a requestQuery. The requestQuery has a 1 to 1 mapping to a query editor row
func (e *cloudWatchExecutor) parseQueries(queries []backend.DataQuery, startTime time.Time, endTime time.Time) (map[string]map[string]*cloudWatchQuery, error) {
	requestQueries := make(map[string]map[string]*cloudWatchQuery)
	migratesQueries, err := migrateLegacyQuery(queries, startTime, endTime)
	if err != nil {
		return nil, err
	}
	for _, query := range migratesQueries {
		model, err := simplejson.NewJson(query.JSON)
		if err != nil {
			return nil, &queryError{err: err, RefID: query.RefID}
		}

		queryType := model.Get("type").MustString()
		if queryType != "timeSeriesQuery" && queryType != "" {
			continue
		}

		refID := query.RefID
		query, err := parseRequestQuery(model, refID, startTime, endTime)
		if err != nil {
			return nil, &queryError{err: err, RefID: refID}
		}

		if _, exist := requestQueries[query.Region]; !exist {
			requestQueries[query.Region] = make(map[string]*cloudWatchQuery, 0)
		}
		requestQueries[query.Region][query.Id] = query
	}

	return requestQueries, nil
}

// migrateLegacyQuery migrates queries that has a `statistics` field to a use the `statistic` field instead.
// This migration is also done in the frontend, so this should only ever be needed for alerting queries
// In case the query used more than one stat, the first stat in the slice will be used in the statistic field
func migrateLegacyQuery(queries []backend.DataQuery, startTime time.Time, endTime time.Time) ([]*backend.DataQuery, error) {
	migratedQueries := []*backend.DataQuery{}
	for _, q := range queries {
		query := q
		model, err := simplejson.NewJson(query.JSON)
		if err != nil {
			return nil, err
		}

		_, err = model.Get("statistic").String()
		// If there's not a statistic property in the json, we now it's the legacy format and then it has to be migrated
		if err != nil {
			stats := model.Get("statistics").MustStringArray()
			model.Del("statistics")
			model.Set("statistic", stats[0])
			query.JSON, err = model.MarshalJSON()
			if err != nil {
				return nil, err
			}
		}

		migratedQueries = append(migratedQueries, &query)
	}

	return migratedQueries, nil
}

func parseRequestQuery(model *simplejson.Json, refId string, startTime time.Time, endTime time.Time) (*cloudWatchQuery, error) {
	plog.Debug("Parsing request query", "query", model)
	reNumber := regexp.MustCompile(`^\d+$`)
	region, err := model.Get("region").String()
	if err != nil {
		return nil, err
	}
	namespace, err := model.Get("namespace").String()
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %v", err)
	}
	metricName, err := model.Get("metricName").String()
	if err != nil {
		return nil, fmt.Errorf("failed to get metricName: %v", err)
	}
	dimensions, err := parseDimensions(model)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dimensions: %v", err)
	}

	statistic, err := model.Get("statistic").String()

	p := model.Get("period").MustString("")
	var period int
	if strings.ToLower(p) == "auto" || p == "" {
		deltaInSeconds := endTime.Sub(startTime).Seconds()
		periods := []int{60, 300, 900, 3600, 21600, 86400}
		datapoints := int(math.Ceil(deltaInSeconds / 2000))
		period = periods[len(periods)-1]
		for _, value := range periods {
			if datapoints <= value {
				period = value
				break
			}
		}
	} else {
		if reNumber.Match([]byte(p)) {
			period, err = strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("failed to parse period as integer: %v", err)
			}
		} else {
			d, err := time.ParseDuration(p)
			if err != nil {
				return nil, fmt.Errorf("failed to parse period as duration: %v", err)
			}
			period = int(d.Seconds())
		}
	}

	id := model.Get("id").MustString("")
	if id == "" {
		id = strings.ToLower(refId)
	}
	expression := model.Get("expression").MustString("")
	alias := model.Get("alias").MustString()
	returnData := !model.Get("hide").MustBool(false)
	queryType := model.Get("type").MustString()
	if queryType == "" {
		// If no type is provided we assume we are called by alerting service, which requires to return data!
		// Note, this is sort of a hack, but the official Grafana interfaces do not carry the information
		// who (which service) called the TsdbQueryEndpoint.Query(...) function.
		returnData = true
	}

	matchExact := model.Get("matchExact").MustBool(true)

	return &cloudWatchQuery{
		RefId:          refId,
		Region:         region,
		Id:             id,
		Namespace:      namespace,
		MetricName:     metricName,
		Statistic:          statistic,
		Expression:     expression,
		ReturnData:     returnData,
		Dimensions:     dimensions,
		Period:         period,
		Alias:          alias,
		MatchExact:     matchExact,
		UsedExpression: "",
		DeepLink:       "",
	}, nil
}

func parseStatistics(model *simplejson.Json) []string {
	var statistics []string
	for _, s := range model.Get("statistics").MustArray() {
		statistics = append(statistics, s.(string))
	}

	return statistics
}

func parseDimensions(model *simplejson.Json) (map[string][]string, error) {
	parsedDimensions := make(map[string][]string)
	for k, v := range model.Get("dimensions").MustMap() {
		// This is for backwards compatibility. Before 6.5 dimensions values were stored as strings and not arrays
		if value, ok := v.(string); ok {
			parsedDimensions[k] = []string{value}
		} else if values, ok := v.([]interface{}); ok {
			for _, value := range values {
				parsedDimensions[k] = append(parsedDimensions[k], value.(string))
			}
		} else {
			return nil, errors.New("unknown type as dimension value")
		}
	}

	sortedDimensions := sortDimensions(parsedDimensions)
	return sortedDimensions, nil
}

func sortDimensions(dimensions map[string][]string) map[string][]string {
	sortedDimensions := make(map[string][]string)
	var keys []string
	for k := range dimensions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		sortedDimensions[k] = dimensions[k]
	}
	return sortedDimensions
}
