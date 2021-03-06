package main

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jetstack/cert-manager/test/acme/dns"
)

var (
	// Environment variable holding the name of the zone to test, ex: example.com however this needs to be a zone you have control over in Loopia.
	// This needs to be set before running the test.
	zone = os.Getenv("TEST_ZONE_NAME")

	// Environment variable for enabling the strict mode testing.
	strictmodeenv = os.Getenv("TEST_STRICT_MODE")
)

func TestRunsSuite(t *testing.T) {
	// The manifest path should contain a file named config.json that is a sniplet of valid configuration that should be included on the ChallengeRequest passed as part of the test cases.
	// The test fixture also requires the kubebuilder-tools binary for your os/architecture to be downloaded to testdata/bin, there is a script supplied that does this in testdata/scripts.

	var strictmode, err = strconv.ParseBool(strictmodeenv)
	if err != nil {
		strictmode = false
	}

	solver := &loopiaDNSProviderSolver{}
	fixture := dns.NewFixture(solver,
		dns.SetStrict(strictmode),
		dns.SetBinariesPath("testdata/bin"),
		dns.SetResolvedZone(zone),
		dns.SetAllowAmbientCredentials(false),
		dns.SetManifestPath("testdata/loopia"),
		dns.SetPollInterval(time.Second*60),
		dns.SetPropagationLimit(time.Minute*30),
	)

	fixture.RunConformance(t)
}
