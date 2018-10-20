package version_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/bborbe/kafka-atlassian-version-collector/avro"
	"github.com/bborbe/kafka-atlassian-version-collector/mocks"
	"github.com/bborbe/kafka-atlassian-version-collector/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Version Fetcher", func() {
	It("work with empty", func() {
		httpClient := &mocks.HttpClient{}
		httpClient.DoReturnsOnCall(0, &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`{}`)),
		}, nil)
		fetcher := &version.Fetcher{
			HttpClient: httpClient,
		}
		versions := make(chan avro.ApplicationVersionAvailable)
		var list []avro.ApplicationVersionAvailable
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for version := range versions {
				list = append(list, version)
			}
		}()
		err := fetcher.Fetch(context.Background(), versions)
		Expect(err).NotTo(HaveOccurred())
		close(versions)
		wg.Wait()
		Expect(list).To(HaveLen(0))
	})
	It("work with empty", func() {
		httpClient := &mocks.HttpClient{}
		httpClient.DoReturnsOnCall(0, &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`{"applications":[{"key":"foo","hostingSupport":{"server":true}}]}`)),
		}, nil)
		httpClient.DoReturnsOnCall(1, &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`{"name":"Banana","versions":[{"version":"1.3.37"}]}`)),
		}, nil)
		fetcher := &version.Fetcher{
			HttpClient: httpClient,
		}
		versions := make(chan avro.ApplicationVersionAvailable)
		var list []avro.ApplicationVersionAvailable
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for version := range versions {
				list = append(list, version)
			}
		}()
		err := fetcher.Fetch(context.Background(), versions)
		close(versions)
		Expect(err).NotTo(HaveOccurred())
		wg.Wait()
		Expect(list).To(HaveLen(1))
		Expect(list[0].App).To(Equal("Banana"))
		Expect(list[0].Version).To(Equal("1.3.37"))
	})
})
