package intersightsdk

import (
	"context"
	"net/http"
	"net/url"
)

func (c *APIClient) PrepareRequest(
	ctx context.Context,
	path string, method string,
	postBody interface{},
	headerParams map[string]string,
	queryParams url.Values,
	formParams url.Values,
	formFiles []formFile,
) (localVarRequest *http.Request, err error) {

	return c.prepareRequest(
		ctx,
		path, method,
		postBody,
		headerParams,
		queryParams,
		formParams,
		formFiles)
}

func (c *APIClient) CallAPI(request *http.Request) (*http.Response, error) {
	return c.callAPI(request)
}

func (c *APIClient) DoGet(ctx context.Context,
	path string, method string, headerParams map[string]string,
	queryParams url.Values,
) (*http.Response, error) {

	request, err := c.prepareRequest(
		ctx,
		path, method,
		nil,
		headerParams,
		queryParams,
		url.Values{},
		[]formFile{},
	)

	if err != nil {
		return nil, err
	}

	return c.callAPI(request)
}

func (c *APIClient) DoPost(ctx context.Context,
	path string, method string,
	postBody interface{},
	headerParams map[string]string,
	queryParams url.Values,
) (*http.Response, error) {

	request, err := c.prepareRequest(
		ctx,
		path, method,
		postBody,
		headerParams,
		queryParams,
		url.Values{},
		[]formFile{},
	)

	if err != nil {
		return nil, err
	}

	return c.callAPI(request)
}
