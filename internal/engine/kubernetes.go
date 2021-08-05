package engine

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/infrahq/infra/internal/logging"
	"github.com/jessevdk/go-flags"
	ipv4 "github.com/signalsciences/ipv4"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
)

const (
	NamespaceFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	CaFilePath        = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	SaFilePath        = "/var/run/secrets/infra-engine-anonymous/token"
)

type Kubernetes struct {
	mu     sync.Mutex
	config *rest.Config
}

func NewKubernetes() (*Kubernetes, error) {
	k := &Kubernetes{}

	config, err := rest.InClusterConfig()
	if err != nil {
		return k, err
	}

	k.config = config

	return k, err
}

func (k *Kubernetes) UpdateRoles(rbs []RoleBinding) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	subjects := make(map[string][]rbacv1.Subject)

	for _, rb := range rbs {
		subjects[rb.Role] = append(subjects[rb.Role], rbacv1.Subject{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "User",
			Name:     rb.User,
		})
	}

	crbs := []*rbacv1.ClusterRoleBinding{}
	for role, subjs := range subjects {
		crbs = append(crbs, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "infra-" + role,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "infra",
				},
			},
			Subjects: subjs,
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     role,
			},
		})
	}

	// Create empty crbs for roles with no users
	clientset, err := kubernetes.NewForConfig(k.config)
	if err != nil {
		return err
	}

	existingCrbs, err := clientset.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{LabelSelector: "app.kubernetes.io/managed-by=infra"})
	if err != nil {
		return err
	}

	// Create or update CRBs for users
	for _, crb := range crbs {
		_, err = clientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), crb, metav1.UpdateOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb, metav1.CreateOptions{})
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	// Delete any CRBs managed by infra that aren't in the config
	var toDelete []rbacv1.ClusterRoleBinding
	for _, e := range existingCrbs.Items {
		var found bool
		for _, crb := range crbs {
			if crb.Name == e.Name {
				found = true
			}
		}

		if !found {
			toDelete = append(toDelete, e)
		}
	}

	for _, td := range toDelete {
		err := clientset.RbacV1().ClusterRoleBindings().Delete(context.TODO(), td.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// Originally from https://github.com/DataDog/datadog-agent
// Apache 2.0 license
func (k *Kubernetes) eksClusterName() (string, error) {
	res, err := http.Get("http://169.254.169.254/latest/dynamic/instance-identity/document")
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", errors.New("received non-OK code from metadata service")
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var identity struct {
		Region     string
		InstanceID string
	}

	err = json.Unmarshal(body, &identity)
	if err != nil {
		return "", err
	}

	awsSess, err := session.NewSession(&aws.Config{
		Region: aws.String(identity.Region),
	})
	if err != nil {
		return "", err
	}

	connection := ec2.New(awsSess)
	ec2Tags, err := connection.DescribeTagsWithContext(context.Background(),
		&ec2.DescribeTagsInput{
			Filters: []*ec2.Filter{{
				Name: aws.String("resource-id"),
				Values: []*string{
					aws.String(identity.InstanceID),
				},
			}},
		},
	)

	if err != nil {
		return "", err
	}

	tags := []string{}
	for _, tag := range ec2Tags.Tags {
		tags = append(tags, fmt.Sprintf("%s:%s", *tag.Key, *tag.Value))
	}

	var clusterName string
	for _, tag := range tags {
		if strings.HasPrefix(tag, "kubernetes.io/cluster/") { // tag key format: kubernetes.io/cluster/clustername"
			key := strings.Split(tag, ":")[0]
			clusterName = strings.Split(key, "/")[2] // rely on ec2 tag format to extract clustername
			break
		}
	}

	if clusterName == "" {
		return "", errors.New("unable to parse cluster name from EC2 tags")
	}

	return clusterName, nil
}

// Originally from https://github.com/DataDog/datadog-agent
// Apache 2.0 license
func (k *Kubernetes) gkeClusterName() (string, error) {
	req, err := http.NewRequest("GET", "http://169.254.169.254/computeMetadata/v1/instance/attributes/cluster-name", nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Metadata-Flavor", "Google")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", errors.New("received non-OK code from metadata service")
	}

	name, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(name), nil
}

// Originally from https://github.com/DataDog/datadog-agent
// Apache 2.0 license
func (k *Kubernetes) aksClusterName() (string, error) {
	req, err := http.NewRequest("GET", "http://169.254.169.254/metadata/instance/compute/resourceGroupName?api-version=2017-08-01&format=text", nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Metadata", "true")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", errors.New("received non-OK code from metadata service")
	}

	all, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error while reading response from azure metadata endpoint: %s", err)
	}

	splitAll := strings.Split(string(all), "_")
	if len(splitAll) < 4 || strings.ToLower(splitAll[0]) != "mc" {
		return "", fmt.Errorf("cannot parse the clustername from resource group name: %s", all)
	}

	return splitAll[len(splitAll)-2], nil
}

func (k *Kubernetes) kopsClusterName() (string, error) {
	// Create empty crbs for roles with no users
	clientset, err := kubernetes.NewForConfig(k.config)
	if err != nil {
		return "", err
	}

	pods, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "k8s-app=kube-controller-manager",
	})
	if err != nil {
		return "", err
	}

	if len(pods.Items) == 0 {
		return "", errors.New("no kube-controller-manager pods to inspect")
	}

	pod := pods.Items[0]
	specContainers := pod.Spec.Containers

	if len(specContainers) == 0 {
		return "", errors.New("no containers in kube-controller-manager podspec")
	}

	container := specContainers[0]

	var opts struct {
		ClusterName string `long:"cluster-name"`
	}
	p := flags.NewParser(&opts, flags.IgnoreUnknown)
	_, err = p.ParseArgs(container.Args)
	if err != nil {
		return "", err
	}

	if len(opts.ClusterName) == 0 {
		return "", errors.New("empty cluster-name argument in kube-controller-manager pod spec")
	}

	return opts.ClusterName, nil
}

func (k *Kubernetes) Name() (string, error) {
	name, err := k.eksClusterName()
	if err == nil {
		return name, nil
	}

	name, err = k.gkeClusterName()
	if err == nil {
		return name, nil
	}

	name, err = k.aksClusterName()
	if err == nil {
		return name, nil
	}

	name, err = k.kopsClusterName()
	if err == nil {
		return name, nil
	}

	logging.L.Debug("could not fetch cluster name, resorting to hashed cluster CA")

	ca, err := k.CA()
	if err != nil {
		return "", err
	}

	h := sha1.New()
	h.Write(ca)
	hash := h.Sum(nil)

	return "cluster-" + hex.EncodeToString(hash)[:8], nil
}

func (k *Kubernetes) Namespace() (string, error) {
	contents, err := ioutil.ReadFile(NamespaceFilePath)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

func (k *Kubernetes) CA() ([]byte, error) {
	contents, err := ioutil.ReadFile(CaFilePath)
	if err != nil {
		return nil, err
	}
	return contents, nil
}

func (k *Kubernetes) SaToken() (string, error) {
	contents, err := ioutil.ReadFile(SaFilePath)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

var EndpointExcludeList = map[string]bool{
	"127.0.0.1":                            true,
	"0.0.0.0":                              true,
	"localhost":                            true,
	"kubernetes.default.svc.cluster.local": true,
	"kubernetes.default.svc.cluster":       true,
	"kubernetes.default.svc":               true,
	"kubernetes.default":                   true,
	"kubernetes":                           true,
	"docker-for-desktop":                   true,
}

func (k *Kubernetes) Endpoint() (string, error) {
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", "kubernetes.default.svc:443", conf)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	certs := conn.ConnectionState().PeerCertificates

	// Determine the cluster endpoint by inspecting the cluster's
	// certificate SAN values
	var dnsNames []string
	for _, cert := range certs {
		dnsNames = append(dnsNames, cert.DNSNames...)
		for _, ip := range cert.IPAddresses {
			dnsNames = append(dnsNames, ip.String())
		}
	}

	var filteredDNSNames []string
	for _, n := range dnsNames {
		// Filter out known
		if EndpointExcludeList[n] {
			continue
		}

		// Filter out private IP addresses
		if ipv4.IsPrivate(n) {
			continue
		}

		// Filter out internal dns names
		if strings.Contains(n, ".internal") {
			continue
		}

		if strings.HasSuffix(n, ".local") {
			continue
		}

		filteredDNSNames = append(filteredDNSNames, n)
	}

	if len(filteredDNSNames) == 0 {
		return "", errors.New("could not determine cluster endpoint")
	}

	return filteredDNSNames[0], nil
}
