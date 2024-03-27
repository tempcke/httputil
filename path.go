package httputil

import (
	"net/url"
	"strings"
)

// Path builds a URL string which can be feed into url.Parse or http.NewRequest
// see the tests for all the different ways you can use it
type Path struct {
	baseURL          string            // ex: http://127.0.0.1:42407
	template         string            // ex: /supplier/:supplierID
	prefix           string            // ex: v1
	paramValMap      map[string]string // ex: {"supplierID": "some-id"}
	queryParamValMap url.Values
}

func (p Path) String() string {
	return p.host() + p.path() + p.query()
}

// NewPath constructs Path
func NewPath(template string) Path {
	if u, err := url.Parse(template); err == nil {
		// strip host if present
		template = u.Path
		if len(u.RawQuery) > 0 {
			template += "?" + u.RawQuery
		}
	}
	return Path{template: template}
}
func (p Path) WithBaseURL(url string) Path {
	p.baseURL = url
	return p
}
func (p Path) WithPrefix(basePath string) Path {
	p.prefix = basePath
	return p
}
func (p Path) WithParam(param, value string) Path {
	return p.WithParams(map[string]string{param: value})
}
func (p Path) WithParams(params map[string]string) Path {
	if p.paramValMap == nil {
		p.paramValMap = make(map[string]string)
	}
	for k, v := range params {
		if k[0:1] != ":" {
			k = ":" + k
		}
		p.paramValMap[k] = v
	}
	return p
}
func (p Path) WithQuery(key string, values ...string) Path {
	if p.queryParamValMap == nil {
		p.queryParamValMap = url.Values{}
	}

	if len(values) == 0 {
		p.queryParamValMap.Set(key, "")
	}

	for _, v := range values {
		if v != "" {
			p.queryParamValMap.Add(key, v)
		}
	}

	return p
}
func (p Path) WithQueryArgs(args map[string]string) Path {
	if p.queryParamValMap == nil {
		p.queryParamValMap = url.Values{}
	}

	for k, v := range args {
		if v != "" {
			p.queryParamValMap.Add(k, v)
		}
	}

	return p
}
func (p Path) WithQueryValues(query url.Values) Path {
	p.queryParamValMap = query
	return p
}

func (p Path) host() string { return p.trim(p.baseURL) }
func (p Path) path() string {
	var (
		path  = p.trim(p.prefix) + "/" + p.trim(p.template)
		elems = strings.Split(path, "/")
	)

	for i, elem := range elems {
		if v, ok := p.paramValMap[elem]; ok {
			elems[i] = v
		}
	}

	return "/" + p.trim(strings.Join(elems, "/"))
}
func (p Path) query() string {
	if len(p.queryParamValMap) > 0 {
		return "?" + p.queryParamValMap.Encode()
	}
	return ""
}
func (p Path) trim(s string) string { return strings.Trim(s, "/") }
