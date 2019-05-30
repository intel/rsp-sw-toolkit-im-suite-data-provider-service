package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

var testPipeLoader = goplumber.NewFSLoader("config/pipelines")
var testTmplLoader = goplumber.NewFSLoader("config/templates")
var testDataLoader = goplumber.NewFSLoader("testdata")
var testMemStore = goplumber.NewMemoryStore()

type overrideStore struct {
	goplumber.PipelineStore
	overrides map[string][]byte
}

func (os overrideStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	r, ok := os.overrides[key]
	if ok {
		logrus.Debugf("using override for %s", key)
		return r, true, nil
	}
	return os.PipelineStore.Get(ctx, key)
}

var testSecretStore = overrideStore{
	PipelineStore: goplumber.NewDockerSecretsStore("testdata"),
	overrides:     map[string][]byte{},
}

func getTestData(w *expect.TWrapper, filename string) []byte {
	w.Helper()
	return w.ShouldHaveResult(testDataLoader.GetFile(filename)).([]byte)
}

func getTestPipeline(w *expect.TWrapper, name, addr string) goplumber.Pipeline {
	w.Helper()
	conf := w.ShouldHaveResult(testPipeLoader.GetFile(name)).([]byte)
	pcon := goplumber.PipelineConnector{
		TemplateLoader: testTmplLoader,
		KVData:         testMemStore,
		Secrets:        testSecretStore,
	}
	testSecretStore.overrides["asnPipelineConfig.json"] =
		[]byte(fmt.Sprintf(`
{
  "siteID": "rrs-gateway",
  "dataEndpoint": "http://example.com/data",
  "cloudConnEndpoint": "%[1]s/cloudconn",
  "coreDataLookup": "%[1]s/core-data",
  "mqttEndpoint": "mosquitto-server:9883",
  "mqttTopics": [ "rfid/gw/shippingmasterdata" ],
  "dataSchemaFile": "ASNSchema.json",
  "oauthConfig": { "useAuth": false }
}
`, addr))
	testSecretStore.overrides["skuPipelineConfig.json"] =
		[]byte(fmt.Sprintf(`
{
  "siteID": "rrs-gateway",
  "dataEndpoint": "http://example.com/data",
  "cloudConnEndpoint": "%[1]s/cloudconn",
  "coreDataLookup": "%[1]s/core-data",
  "mqttEndpoint": "mosquitto-server:9883",
  "mqttTopics": [ "rfid/gw/productmasterdata" ],
  "dataSchemaFile": "SKUSchema.json",
  "oauthConfig": { "useAuth": false }
}
`, addr))
	return w.ShouldHaveResult(goplumber.NewPipeline(conf, pcon)).(goplumber.Pipeline)
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

	r, ok, err := testMemStore.Get(ctx, memname)
	w.ShouldHaveResult(r, err)
	w.ShouldBeTrue(ok)
	rn := w.ShouldHaveResult(strconv.Atoi(string(r))).(int)
	w.Logf("%d", rn)

	// run a second time
	w.ShouldSucceed(p.Execute(context.Background()))
	r2, ok, err := testMemStore.Get(ctx, memname)
	w.ShouldHaveResult(r2, err)
	w.ShouldBeTrue(ok)
	rn2 := w.ShouldHaveResult(strconv.Atoi(string(r2))).(int)
	w.Logf("%d", rn2)
	w.ShouldBeTrue(rn2 >= rn)
}

func TestSKU(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()
	skuData := base64.StdEncoding.EncodeToString(getTestData(w, "skuData.json"))

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
		p := getTestPipeline(w, "SKUPipeline.json", addr)
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
		p := getTestPipeline(w, "ASNPipeline.json", addr)
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
