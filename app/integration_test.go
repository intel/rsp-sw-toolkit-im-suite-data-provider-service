// +build integration

package app

import (
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/goplumber"
	"github.impcloud.net/Responsive-Retail-Inventory/expect"
	"testing"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func getIntegrationTestPipeline(w *expect.TWrapper, name string) goplumber.Pipeline {
	w.Helper()
	conf := getTestData(w, name)
	pcon := goplumber.PipelineConnector{
		TemplateLoader: testTmplLoader,
		KVData:         testMemStore,
		Secrets:        goplumber.NewDockerSecretsStore("config/testdata"),
	}
	/*
		w.ShouldSucceed(testMemStore.Put(context.Background(), "cloudconn.endpoint",
			[]byte(`"http://192.168.99.100:9004"`)))
	*/
	return w.ShouldHaveResult(goplumber.NewPipeline(conf, pcon)).(goplumber.Pipeline)
}

func TestSKUIntegration(t *testing.T) {
	// this test pulls 'SKU' data from an endpoint and publish it to MQTT
	w := expect.WrapT(t).StopOnMismatch()
	p := getIntegrationTestPipeline(w, "SKUPipeline.json")
	testPipeline(w, &p, "skus.lastUpdated")
}

func TestASNIntegration(t *testing.T) {
	w := expect.WrapT(t).StopOnMismatch()
	p := getIntegrationTestPipeline(w, "ASNPipeline.json")
	testPipeline(w, &p, "asn.lastUpdated")
}

func BenchmarkSKU(b *testing.B) {
	w := expect.WrapT(b).StopOnMismatch()
	logrus.SetLevel(logrus.ErrorLevel)
	p := getIntegrationTestPipeline(w, "SKUPipeline.json")
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		testPipeline(w, &p, "skus.lastUpdated")
	}
}
