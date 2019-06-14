package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/expect"
	"github.impcloud.net/RSP-Inventory-Suite/goplumber"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func getTestData(w *expect.TWrapper, filename string) []byte {
	w.Helper()
	return w.ShouldHaveResult(dataLoader.GetFile(filename)).([]byte)
}

var pipeLoader = goplumber.NewFSLoader("config/pipelines")
var tmplLoader = goplumber.NewFSLoader("config/templates")
var dataLoader = goplumber.NewFSLoader("testdata")
var memoryStore = goplumber.NewMemoryStore()

type multiSearchSource struct {
	sources []goplumber.DataSource
}

func (mss *multiSearchSource) Get(ctx context.Context, key string) ([]byte, bool, error) {
	for i, s := range mss.sources {
		if val, ok, err := s.Get(ctx, key); ok && err == nil {
			return val, true, nil
		} else if err != nil {
			logrus.WithError(err).
				Errorf("failed to get value for key '%s' from source %d",
					key, i)
		}
	}
	return nil, false, errors.Errorf("key '%s' not found in any source", key)
}

func getTestPlumber() goplumber.Plumber {
	p := goplumber.NewPlumber(tmplLoader)
	mss := &multiSearchSource{sources: []goplumber.DataSource{memoryStore, dataLoader}}
	p.TaskGenerators["secret"] = goplumber.NewLoadTaskGenerator(mss)
	p.TaskGenerators["get"] = goplumber.NewLoadTaskGenerator(memoryStore)
	p.TaskGenerators["put"] = goplumber.NewStoreTaskGenerator(memoryStore)

	return p
}

func getTestPipeline(w *expect.TWrapper, plumber goplumber.Plumber, name string) goplumber.Pipeline {
	w.Helper()
	conf := w.ShouldHaveResult(pipeLoader.GetFile(name)).([]byte)
	pipeline := w.ShouldHaveResult(plumber.NewPipeline(conf)).(goplumber.Pipeline)
	return pipeline
}

func withDataServer(w *expect.TWrapper, dataMap map[string][]byte, f func(url string)) map[string][]byte {
	w.Helper()
	callMap := map[string][]byte{}
	s := httptest.NewServer(
		http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer w.ShouldSucceedLater(r.Body.Close)
			callMap[r.URL.Path] = w.ShouldHaveResult(ioutil.ReadAll(r.Body)).([]byte)

			data, ok := dataMap[r.URL.Path]
			if !ok {
				w.Errorf("missing endpoint for %s", r.URL.Path)
				rw.WriteHeader(404)
				return
			}

			w.ShouldHaveResult(rw.Write(data))
			rw.WriteHeader(200)
		}))
	f(s.URL)
	s.Close()
	return callMap
}

func testPipeline(w *expect.TWrapper, p *goplumber.Pipeline, memname string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()
	w.ShouldSucceed(p.Execute(ctx))

	r, ok, err := memoryStore.Get(ctx, memname)
	w.ShouldHaveResult(r, err)
	w.ShouldBeTrue(ok)
	rn := w.ShouldHaveResult(strconv.Atoi(string(r))).(int)
	w.Logf("%d", rn)

	// run a second time
	w.ShouldSucceed(p.Execute(context.Background()))
	r2, ok, err := memoryStore.Get(ctx, memname)
	w.ShouldHaveResult(r2, err)
	w.ShouldBeTrue(ok)
	rn2 := w.ShouldHaveResult(strconv.Atoi(string(r2))).(int)
	w.Logf("%d", rn2)
	w.ShouldBeTrue(rn2 >= rn)
}

func TestSKU(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()
	skuData := base64.StdEncoding.EncodeToString(getTestData(w, "skuData.json"))

	plumber := getTestPlumber()

	dataMap := map[string][]byte{
		"/cloudconn":    []byte(fmt.Sprintf(`{"statuscode":200,"body":"%s"}`, skuData)),
		"/api/v1/event": []byte(``), // POSTed to
	}
	coreData := `"ID":"3325fece-83ca-8736-bc88-bda1d9d56caf","Node":"edgex-core-consul","Address":"127.0.0.1","Datacenter":"dc1","TaggedAddresses":{"lan":"127.0.0.1","wan":"127.0.0.1"},"NodeMeta":{"consul-network-segment":""},"ServiceID":"edgex-core-data","ServiceName":"edgex-core-data","ServiceTags":[],"ServiceMeta":{},"ServiceEnableTagOverride":false,"CreateIndex":15,"ModifyIndex":15`
	results := withDataServer(w, dataMap, func(addr string) {
		serverURL := w.ShouldHaveResult(url.Parse(addr)).(*url.URL)
		dataMap["/core-data"] = []byte(fmt.Sprintf(
			`[{"ServicePort":"%s","ServiceAddress":"%s",%s}]`,
			serverURL.Port(), serverURL.Hostname(), coreData))

		_ = memoryStore.Put(context.Background(), "skuPipelineConfig.json",
			[]byte(fmt.Sprintf(`{
			  "siteID": "rrs-gateway",
			  "dataEndpoint": "http://example.com/data",
			  "cloudConnEndpoint": "%[1]s/cloudconn",
			  "coreDataLookup": "%[1]s/core-data",
			  "mqttEndpoint": "mosquitto-server:9883",
			  "dataSchemaFile": "SKUSchema.json",
			  "oauthConfig": { "useAuth": false }
			}`, addr)))

		p := getTestPipeline(w, plumber, "SKUPipeline.json")
		testPipeline(w, &p, "skus.lastUpdated")
	})

	w.ShouldContain(results, []string{"/api/v1/event", "/cloudconn"})

	type edgexReading struct {
		Name  string
		Value string
	}
	type edgexEvent struct {
		Origin   int
		Device   string
		Readings []edgexReading
	}
	var ee edgexEvent
	w.Logf("%s", results["/api/v1/event"])
	w.ShouldSucceed(json.Unmarshal(results["/api/v1/event"], &ee))
	w.ShouldBeTrue(ee.Origin > 0)
	w.ShouldBeEqual(ee.Device, "SKU_Data_Device")
	w.ShouldHaveLength(ee.Readings, 1)
	w.ShouldBeEqual(ee.Readings[0].Name, "SKU_data")
	w.ShouldNotBeEmptyStr(ee.Readings[0].Value)
}

func TestASN(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()
	asnData := base64.StdEncoding.EncodeToString(getTestData(w, "asnData.json"))

	plumber := getTestPlumber()

	dataMap := map[string][]byte{
		"/cloudconn":    []byte(fmt.Sprintf(`{"statuscode":200,"body":"%s"}`, asnData)),
		"/api/v1/event": []byte(``), // POSTed to
	}
	coreData := `"ID":"3325fece-83ca-8736-bc88-bda1d9d56caf","Node":"edgex-core-consul","Address":"127.0.0.1","Datacenter":"dc1","TaggedAddresses":{"lan":"127.0.0.1","wan":"127.0.0.1"},"NodeMeta":{"consul-network-segment":""},"ServiceID":"edgex-core-data","ServiceName":"edgex-core-data","ServiceTags":[],"ServiceMeta":{},"ServiceEnableTagOverride":false,"CreateIndex":15,"ModifyIndex":15`
	results := withDataServer(w, dataMap, func(addr string) {
		serverURL := w.ShouldHaveResult(url.Parse(addr)).(*url.URL)
		dataMap["/core-data"] = []byte(fmt.Sprintf(
			`[{"ServicePort":"%s","ServiceAddress":"%s",%s}]`,
			serverURL.Port(), serverURL.Hostname(), coreData))

		_ = memoryStore.Put(context.Background(), "asnPipelineConfig.json",
			[]byte(fmt.Sprintf(`{
				  "siteID": "rrs-gateway",
				  "dataEndpoint": "http://example.com/data",
				  "cloudConnEndpoint": "%[1]s/cloudconn",
				  "coreDataLookup": "%[1]s/core-data",
				  "mqttEndpoint": "mosquitto-server:9883",
				  "dataSchemaFile": "ASNSchema.json",
				  "oauthConfig": { "useAuth": false }
				}`, addr)))

		p := getTestPipeline(w, plumber, "ASNPipeline.json")
		testPipeline(w, &p, "asn.lastUpdated")
	})

	w.ShouldContain(results, []string{"/api/v1/event", "/cloudconn"})

	type edgexReading struct {
		Name  string
		Value string
	}
	type edgexEvent struct {
		Origin   int
		Device   string
		Readings []edgexReading
	}
	var ee edgexEvent
	w.Logf("%s", results["/api/v1/event"])
	w.ShouldSucceed(json.Unmarshal(results["/api/v1/event"], &ee))
	w.ShouldBeTrue(ee.Origin > 0)
	w.ShouldBeEqual(ee.Device, "ASN_Data_Device")
	w.ShouldHaveLength(ee.Readings, 1)
	w.ShouldBeEqual(ee.Readings[0].Name, "ASN_data")
	w.ShouldNotBeEmptyStr(ee.Readings[0].Value)
}
