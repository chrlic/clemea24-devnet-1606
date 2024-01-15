package contextdb

import (
	"net/url"
	"strings"
)

func (m *Contexts) lookup(queryUri string) ([]interface{}, error) {
	result := []interface{}{}
	query, err := url.Parse(queryUri)
	if err != nil {
		return result, err
	}
	extension := query.Host
	path := strings.Split(query.Path, "/")
	if len(path) != 2 {
		return result, err
	}
	table := path[0]
	index := path[1]
	params, err := url.ParseQuery(query.RawQuery)
	if err != nil {
		return result, err
	}
	field := query.Fragment

	m.logger.Sugar().Debugf("Looking up extension '%s', table '%s', by index '%s' with params '%v' optional field '%s'",
		extension, table, index, params, field)

	return result, nil

}
