// +build integration

package app

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/goplumber"
	"github.impcloud.net/Responsive-Retail-Inventory/expect"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

var testPipeLoader = goplumber.NewFSLoader("config/pipelines")
var testTmplLoader = goplumber.NewFSLoader("config/templates")
var testDockerSecretsStore = goplumber.NewDockerSecretsStore("config/testdata")
var testMemStore = goplumber.NewMemoryStore()

func getTestData(w *expect.TWrapper, filename string) []byte {
	w.Helper()
	return w.ShouldHaveResult(testPipeLoader.GetFile(filename)).([]byte)
}

func getTestPipeline(w *expect.TWrapper, name string) goplumber.Pipeline {
	w.Helper()
	conf := getTestData(w, name)
	pcon := goplumber.PipelineConnector{
		TemplateLoader: testTmplLoader,
		KVData:         testMemStore,
		Secrets:        testDockerSecretsStore,
	}
	/*
		w.ShouldSucceed(testMemStore.Put(context.Background(), "cloudconn.endpoint",
			[]byte(`"http://192.168.99.100:9004"`)))
	*/
	return w.ShouldHaveResult(goplumber.NewPipeline(conf, pcon)).(goplumber.Pipeline)
}

func withSiteDataServer(w *expect.TWrapper, port int) (cleanup func()) {
	w.Helper()
	s := httptest.NewUnstartedServer(
		http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			w.ShouldHaveResult(rw.Write([]byte(`{
				"data": {
					"hostName":"px-aaeea0542017",
					"osPlatformName":"RANCHER",
					"portalUrl":"https://portal.pxirr.com/",
					"domainName":"pxirr.com",
					"deviceId":"f932a2ee2b9546b7ba6e824e3916ed14",
					"deviceName":"CH6LabTest002",
					"portalAuthUrl":"https://portalauth.pxirr.com/"
				}
			}`)))
			rw.WriteHeader(200)
		}))
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	w.ShouldSucceed(err)
	s.Listener = l
	s.StartTLS()
	return func() { s.Close() }
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
	// this test pulls 'SKU' data from an endpoint and publish it to MQTT
	w := expect.WrapT(t).StopOnMismatch()
	p := getTestPipeline(w, "SKUPipeline.json")
	defer withSiteDataServer(w, 54321)()
	testPipeline(w, &p, "skus.lastUpdated")
}

func TestASN(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()
	p := getTestPipeline(w, "ASNPipeline.json")
	defer withSiteDataServer(w, 54321)()
	testPipeline(w, &p, "asn.lastUpdated")
}

func BenchmarkSKU(b *testing.B) {
	w := expect.WrapT(b).StopOnMismatch()
	logrus.SetLevel(logrus.ErrorLevel)
	p := getTestPipeline(w, "SKUPipeline.json")
	defer withSiteDataServer(w, 54321)()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		testPipeline(w, &p, "skus.lastUpdated")
	}
}
