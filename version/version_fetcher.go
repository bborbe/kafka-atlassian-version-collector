// Copyright (c) 2018 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bborbe/kafka-atlassian-version-collector/avro"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

//go:generate counterfeiter -o ../mocks/http_client.go --fake-name HttpClient . httpClient
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Fetcher struct {
	HttpClient httpClient
}

func (v *Fetcher) Fetch(ctx context.Context, versions chan<- avro.ApplicationVersionAvailable) error {
	url := "https://marketplace.atlassian.com/rest/1.0/applications"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrap(err, "build request failed")
	}
	glog.V(1).Infof("%s %s", req.Method, req.URL.String())
	resp, err := v.HttpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}
	if resp.StatusCode/100 != 2 {
		return errors.New("request status code != 2xx")
	}
	var data struct {
		Applications []struct {
			Key            string `json:"key"`
			HostingSupport struct {
				Server bool `json:"server"`
			} `json:"hostingSupport"`
		} `json:"applications"`
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return errors.Wrap(err, "decode json failed")
	}
	for _, application := range data.Applications {
		if !application.HostingSupport.Server {
			continue
		}
		if err := v.fetchApplication(ctx, application.Key, versions); err != nil {
			return err
		}
	}
	return nil
}

func (v *Fetcher) fetchApplication(ctx context.Context, name string, versions chan<- avro.ApplicationVersionAvailable) error {
	url := fmt.Sprintf("https://marketplace.atlassian.com/rest/1.0/applications/%s", name)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrap(err, "build request failed")
	}
	glog.V(1).Infof("%s %s", req.Method, req.URL.String())
	resp, err := v.HttpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}
	if resp.StatusCode/100 != 2 {
		return errors.New("request status code != 2xx")
	}
	var data struct {
		Name     string `json:"name"`
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return errors.Wrap(err, "decode json failed")
	}
	glog.V(3).Infof("found %d tags", len(data.Versions))
	for _, version := range data.Versions {
		glog.V(3).Infof("send version %s %s to channel", data.Name, version.Version)
		select {
		case <-ctx.Done():
			glog.Infof("context done => return")
			return nil
		case versions <- avro.ApplicationVersionAvailable{
			App:     data.Name,
			Version: version.Version,
		}:
		}
	}
	glog.V(3).Infof("send all versions to channel => return")
	return nil
}
