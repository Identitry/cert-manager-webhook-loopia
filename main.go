package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	loopia "github.com/jonlil/loopia-go"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	LoopiaMinTtl = 300 // Loopia requires a TTL of minimum 300 sec.
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	// This will register our loopia DNS provider with the webhook serving  library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single webhook, where the Name() method will be used to disambiguate between the different implementations.
	cmd.RunWebhookServer(GroupName,
		&loopiaDNSProviderSolver{},
	)
}

// loopiaDNSProviderSolver implements the provider-specific logic needed to 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/jetstack/cert-manager/pkg/acme/webhook.Solver` interface.
type loopiaDNSProviderSolver struct {
	client *kubernetes.Clientset
}

// Type holding credential.
type credential struct {
	Username string
	Password string
}

// loopiaDNSProviderConfig is a structure that is used to decode into when solving a DNS01 challenge.
// This information is provided by cert-manager, and may be a reference to additional configuration that's needed to solve the challenge for this particular certificate or issuer.
// This typically includes references to Secret resources containing DNS provider credentials, in cases where a 'multi-tenant' DNS solver is being created.
// If you do *not* require per-issuer or per-certificate configuration to be provided to your webhook, you can skip decoding altogether in favour of using CLI flags or similar to provide configuration.
// You should not include sensitive information here. If credentials need to be used by your provider here, you should reference a Kubernetes Secret resource and fetch these credentials using a Kubernetes clientset.
type loopiaDNSProviderConfig struct {
	// These fields will be set by users in the `issuer.spec.acme.dns01.providers.webhook.config` field.
	UsernameSecretKeyRef cmmeta.SecretKeySelector `json:"usernameSecretKeyRef"`
	PasswordSecretKeyRef cmmeta.SecretKeySelector `json:"passwordSecretKeyRef"`
}

// Name is used as the name for this DNS solver when referencing it on the ACME Issuer resource.
// This should be unique **within the group name**, i.e. you can have two solvers configured with the same Name()
// **so long as they do not co-exist within a single webhook deployment**. For example, `cloudflare` may be used as the name of a solver.
func (c *loopiaDNSProviderSolver) Name() string {
	return "loopia"
}

// Present is responsible for actually presenting the DNS record with the DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the solver has correctly configured the DNS provider.
func (c *loopiaDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	klog.V(6).Infof("call function Present: namespace=%s, zone=%s, fqdn=%s", ch.ResourceNamespace, ch.ResolvedZone, ch.ResolvedFQDN)

	// LOad config.
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}
	klog.V(6).Infof("decoded configuration %v", cfg)

	// Get credentials for connecting to Loopia.
	creds, err := c.getCredentials(&cfg, ch.ResourceNamespace)
	if err != nil {
		return fmt.Errorf("unable to get credential: %v", err)
	}

	// Initialize new Loopia client.
	loopiaClient, err := loopia.New(creds.Username, creds.Password)
	if err != nil {
		return fmt.Errorf("could not initialize loopia client: %v", err)
	}

	// Split and format domain and sub domain values.
	subdomain, domain := c.getDomainAndSubdomain(ch)
	klog.V(6).Infof("present for subdomain=%s, domain=%s", subdomain, domain)

	// Get loopia records for subdomain.
	zoneRecords, err := loopiaClient.GetZoneRecords(domain, subdomain)
	if err != nil {
		return fmt.Errorf("unable tot ge zone records: %v", err)
	}

	// Exit if record is already present by type and value.
	for _, zoneRecord := range zoneRecords {
		if zoneRecord.Type == "TXT" && zoneRecord.Value == ch.Key {
			return nil
		}
	}

	// Record isn't present, instantiate it.
	record := loopia.Record{
		TTL:      LoopiaMinTtl,
		Type:     "TXT",
		Value:    ch.Key,
		Priority: 0,
	}

	// Record isn't present, create it.
	err = loopiaClient.AddZoneRecord(domain, subdomain, &record)
	if err != nil {
		return fmt.Errorf("unable to create TXT record: %v", err)
	}

	return nil
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (c *loopiaDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	klog.V(6).Infof("call function CleanUp: namespace=%s, zone=%s, fqdn=%s", ch.ResourceNamespace, ch.ResolvedZone, ch.ResolvedFQDN)

	// Load config.
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}

	// Get credentials for connecting to Loopia.
	creds, err := c.getCredentials(&cfg, ch.ResourceNamespace)
	if err != nil {
		return fmt.Errorf("unable to get credential: %v", err)
	}

	// Initialize Loopia client.
	loopiaClient, err := loopia.New(creds.Username, creds.Password)
	if err != nil {
		return fmt.Errorf("could not initialize loopia client: %v", err)
	}

	// Split and format domain and sub domain values.
	subdomain, domain := c.getDomainAndSubdomain(ch)
	klog.V(6).Infof("cleanup for subdomain=%s, domain=%s", subdomain, domain)

	// Get loopia records for subdomain.
	zoneRecords, err := loopiaClient.GetZoneRecords(domain, subdomain)
	if err != nil {
		return fmt.Errorf("unable to get zone records: %v", err)
	}

	// Loop records to get the one to delete, if found delete it...
	for _, zoneRecord := range zoneRecords {
		if zoneRecord.Type == "TXT" && zoneRecord.Value == ch.Key {
			_, err := loopiaClient.RemoveZoneRecord(domain, subdomain, zoneRecord.ID)
			if err != nil {
				return fmt.Errorf("unable to delete TXT record: %v", err)
			}
		}
	}

	// Clean up subdomain.
	if len(zoneRecords) <= 1 {
		_, err := loopiaClient.RemoveSubDomain(domain, subdomain)
		if err != nil {
			return fmt.Errorf("unable to remove subdomain: %v", err)
		}
	}

	return nil
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes client that can be used to fetch resources from the Kubernetes API, e.g.
// * Secret resources containing credentials used to authenticate with DNS
// * provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases where a SIGTERM or similar signal is sent to the webhook process.
func (c *loopiaDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, _ <-chan struct{}) error {
	klog.V(6).Infof("call function Initialize")
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return fmt.Errorf("unable to get k8s client: %v", err)
	}
	c.client = cl
	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (loopiaDNSProviderConfig, error) {
	cfg := loopiaDNSProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

// Split and format domain and sub domain values.
func (c *loopiaDNSProviderSolver) getDomainAndSubdomain(ch *v1alpha1.ChallengeRequest) (string, string) {
	// ch.ResolvedZone form: example.com.
	// ch.ResolvedFQDN form:  _acme-challenge.example.com.
	// Both ch.ResolvedZone and ch.ResolvedFQDN end with a dot: '.'
	subDomain := strings.TrimSuffix(ch.ResolvedFQDN, ch.ResolvedZone)
	subDomain = strings.TrimSuffix(subDomain, ".")
	domain := strings.TrimSuffix(ch.ResolvedZone, ".")
	return subDomain, domain
}

// Get Loopia API credentials from Kubernetes secret.
func (c *loopiaDNSProviderSolver) getCredentials(cfg *loopiaDNSProviderConfig, namespace string) (*credential, error) {
	creds := credential{}

	// Get Username.
	klog.V(6).Infof("try to load secret `%s` with key `%s`", cfg.UsernameSecretKeyRef.Name, cfg.UsernameSecretKeyRef.Key)
	usernameSecret, err := c.client.CoreV1().Secrets(namespace).Get(context.Background(), cfg.UsernameSecretKeyRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to load secret %q: %s", namespace+"/"+cfg.UsernameSecretKeyRef.Name, err.Error())
	}
	if username, ok := usernameSecret.Data[cfg.UsernameSecretKeyRef.Key]; ok {
		creds.Username = string(username)
	} else {
		return nil, fmt.Errorf("no key %q in secret %q", cfg.UsernameSecretKeyRef, namespace+"/"+cfg.UsernameSecretKeyRef.Name)
	}

	// Get Password.
	klog.V(6).Infof("try to load secret `%s` with key `%s`", cfg.PasswordSecretKeyRef.Name, cfg.PasswordSecretKeyRef.Key)
	passwordSecret, err := c.client.CoreV1().Secrets(namespace).Get(context.Background(), cfg.PasswordSecretKeyRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to load secret %q: %s", namespace+"/"+cfg.PasswordSecretKeyRef.Name, err.Error())
	}
	if password, ok := passwordSecret.Data[cfg.PasswordSecretKeyRef.Key]; ok {
		creds.Password = string(password)
	} else {
		return nil, fmt.Errorf("no key %q in secret %q", cfg.PasswordSecretKeyRef, namespace+"/"+cfg.PasswordSecretKeyRef.Name)
	}

	return &creds, nil
}
