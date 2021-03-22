package main

import (
	"os"
	"testing"
	"time"

	"github.com/jetstack/cert-manager/test/acme/dns"
)

var (
	// Environment variable holding the name of the zone to test, ex: example.com however this needs to be a zone you have control over in Loopia.
	// This needs to be set before running the test.
	zone = os.Getenv("TEST_ZONE_NAME")
)

func TestRunsSuite(t *testing.T) {
	// The manifest path should contain a file named config.json that is a sniplet of valid configuration that should be included on the ChallengeRequest passed as part of the test cases.
	// The test fixture also requires the kubebuilder-tools binary for your os/architecture to be downloaded to testdata/bin, there is a script supplied that does this in testdata/scripts.

	solver := &loopiaDNSProviderSolver{}
	fixture := dns.NewFixture(solver,
		dns.SetBinariesPath("testdata/bin"),
		dns.SetResolvedZone(zone),
		dns.SetAllowAmbientCredentials(false),
		dns.SetManifestPath("testdata/loopia"),
		dns.SetDNSServer("93.188.0.20:53"),
		//dns.SetUseAuthoritative(true),
		dns.SetPollInterval(time.Second*15),
		dns.SetPropagationLimit(time.Minute*30),
	)

	fixture.RunConformance(t)
}
