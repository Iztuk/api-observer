package views

import "net/url"

func parseQueryParams(rawQuery string) url.Values {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return url.Values{}
	}

	return values
}
