package kubernetes

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
	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/jessevdk/go-flags"
	ipv4 "github.com/signalsciences/ipv4"
	"go.uber.org/zap"
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

type RoleBinding struct {
	Role  string
	Users []string
}

// namespaceRole is used as a tuple to pair namespaces and roles as a map key
type namespaceRole struct {
	namespace string
	role      string
}

// rbIdentifier is used to compare RoleBindings by their unique fields
type rbIdentifier struct {
	namespace string
	name      string
}

type Kubernetes struct {
	mu           sync.Mutex
	Config       *rest.Config
	SecretReader SecretReader
}

func NewKubernetes() (*Kubernetes, error) {
	k := &Kubernetes{}

	config, err := rest.InClusterConfig()
	if err != nil {
		return k, err
	}

	k.Config = config

	namespace, err := k.Namespace()
	if err != nil {
		return k, err
	}
	k.SecretReader = NewSecretReader(namespace)

	return k, err
}

// updateRoleBindings generates RoleBindings for ClusterRoles within a specific namespace
func (k *Kubernetes) updateRoleBindings(subjects map[namespaceRole][]rbacv1.Subject) error {
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return err
	}

	// keep the local config clean, use the local cluster roles to check if a binding is actually needed
	crs, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	validCR := make(map[string]bool)
	for _, cr := range crs.Items {
		validCR[cr.Name] = true
	}

	// create the namespaced Role bindings for all the users of each namespaced ClusterRole assignments
	rbs := []*rbacv1.RoleBinding{}
	for nsr, subjs := range subjects {
		if validCR[nsr.role] {
			rbs = append(rbs, &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "infra-" + nsr.role,
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "infra",
					},
					Namespace: nsr.namespace,
				},
				Subjects: subjs,
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     nsr.role,
				},
			})
		} else {
			logging.L.Debug("RoleBinding skipped, ClusterRole does not exist with name: " + nsr.role)
		}
	}

	// passing an empty string to rolebindings for the namespace returns all rolebindings
	existingInfraRbs, err := clientset.RbacV1().RoleBindings("").List(context.TODO(), metav1.ListOptions{LabelSelector: "app.kubernetes.io/managed-by=infra"})
	if err != nil {
		return err
	}
	toDelete := make(map[rbIdentifier]rbacv1.RoleBinding)
	for _, existingRb := range existingInfraRbs.Items {
		rbID := rbIdentifier{
			namespace: existingRb.Namespace,
			name:      existingRb.Name,
		}
		toDelete[rbID] = existingRb
	}

	// Create or update RoleBindings for users
	for _, rb := range rbs {
		_, err = clientset.RbacV1().RoleBindings(rb.Namespace).Update(context.TODO(), rb, metav1.UpdateOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				_, err = clientset.RbacV1().RoleBindings(rb.Namespace).Create(context.TODO(), rb, metav1.CreateOptions{})
				if err != nil {
					if k8sErrors.IsNotFound(err) {
						// the namespace does not exist
						// we can proceed in this case, the role mapping is just not applicable to this cluster
						logging.L.Debug("Skipping unapplicable namespace for this cluster", zap.Error(err))
						continue
					}
					return err
				}
			} else {
				return err
			}
		}
		// remove anything we update or create from the previous RoleBindings that will be deleted
		delete(toDelete, rbIdentifier{namespace: rb.Namespace, name: rb.Name})
	}

	// Delete any RoleBindings managed by infra that aren't in the config
	// Do not need to worry about deleted namespaces as they will also delete all their resources
	for _, td := range toDelete {
		err := clientset.RbacV1().RoleBindings(td.Namespace).Delete(context.TODO(), td.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateClusterRoleBindings generates ClusterRoleBindings for RoleMappings
func (k *Kubernetes) updateClusterRoleBindings(subjects map[string][]rbacv1.Subject) error {
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
	clientset, err := kubernetes.NewForConfig(k.Config)
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

// UpdateRoles converts API roles to role-bindings in the current cluster
func (k *Kubernetes) UpdateRoles(roles []api.Role) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	// group together all users with the same role/namespace permissions
	rbSubjects := make(map[namespaceRole][]rbacv1.Subject) // role-bindings
	// TODO: add namespaces to cluster-roles
	// crbSubjects := make(map[namespaceRole][]rbacv1.Subject) // cluster role-bindings
	crbSubjects := make(map[string][]rbacv1.Subject)
	for _, r := range roles {
		switch r.Kind {
		case api.ROLE:
			if r.Namespace == "" {
				logging.L.Error("skipping role binding with no namespace: " + r.Name)
				continue
			}
			nspaceRole := namespaceRole{
				namespace: r.Namespace,
				role:      r.Name,
			}
			for _, u := range r.Users {
				rbSubjects[nspaceRole] = append(rbSubjects[nspaceRole], rbacv1.Subject{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "User",
					Name:     u.Email,
				})
			}
		case api.CLUSTER_ROLE:
			// TODO
			// nspaceClusterRole := namespaceRole{
			// 	namespace: r.Namespace,
			// 	role:      r.Name,
			// }
			for _, u := range r.Users {
				crbSubjects[r.Name] = append(crbSubjects[r.Name], rbacv1.Subject{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "User",
					Name:     u.Email,
				})
			}
		default:
			logging.L.Error("Unknown role binding kind: " + fmt.Sprintf("%v", r.Kind))
		}
	}

	err := k.updateClusterRoleBindings(crbSubjects)
	if err != nil {
		return err
	}
	err = k.updateRoleBindings(rbSubjects)
	return err
}

func (k *Kubernetes) ec2ClusterName() (string, error) {
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
	name := "instance-id"
	value := identity.InstanceID
	describeInstancesOutput, err := connection.DescribeInstances(&ec2.DescribeInstancesInput{Filters: []*ec2.Filter{{Name: &name, Values: []*string{&value}}}})
	if err != nil {
		return "", err
	}

	reservations := describeInstancesOutput.Reservations
	if len(reservations) == 0 {
		return "", errors.New("could not fetch ec2 instance reservations")
	}

	ec2Instances := reservations[0].Instances
	if len(ec2Instances) == 0 {
		return "", errors.New("could not fetch ec2 instances")
	}

	instance := ec2Instances[0]

	tags := []string{}
	for _, tag := range instance.Tags {
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

func (k *Kubernetes) kubeControllerManagerClusterName() (string, error) {
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return "", err
	}

	k8sAppPods, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "k8s-app=kube-controller-manager",
	})
	if err != nil {
		return "", err
	}

	componentPods, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "component=kube-controller-manager",
	})
	if err != nil {
		return "", err
	}

	pods := append(k8sAppPods.Items, componentPods.Items...)

	if len(pods) == 0 {
		return "", errors.New("no kube-controller-manager pods to inspect")
	}

	pod := pods[0]

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

	if opts.ClusterName == "" {
		return "", errors.New("empty cluster-name argument in kube-controller-manager pod spec")
	}

	return opts.ClusterName, nil
}

func (k *Kubernetes) Name() (string, error) {
	name, err := k.ec2ClusterName()
	if err == nil {
		return name, nil
	}

	logging.L.Debug("could not fetch ec2 cluster name: " + err.Error())

	name, err = k.gkeClusterName()
	if err == nil {
		return name, nil
	}

	logging.L.Debug("could not fetch gke cluster name: " + err.Error())

	name, err = k.aksClusterName()
	if err == nil {
		return name, nil
	}

	logging.L.Debug("could not fetch aks cluster name: " + err.Error())

	name, err = k.kubeControllerManagerClusterName()
	if err == nil {
		return name, nil
	}

	logging.L.Debug("could not fetch kube-controller-manager cluster name: " + err.Error())

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

var EndpointExclude = map[string]bool{
	"127.0.0.1":                            true,
	"0.0.0.0":                              true,
	"localhost":                            true,
	"kubernetes.default.svc.cluster.local": true,
	"kubernetes.default.svc.cluster":       true,
	"kubernetes.default.svc":               true,
	"kubernetes.default":                   true,
	"kubernetes":                           true,
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
		// Filter out reserved kubernetes domain names
		if EndpointExclude[n] {
			continue
		}

		// Filter out private IP addresses
		if ipv4.IsPrivate(n) {
			continue
		}

		// Filter out hostnames that aren't IP addresses or domain names
		if !strings.Contains(n, ".") {
			continue
		}

		// Filter out internal dns names
		if strings.Contains(n, ".internal") {
			continue
		}

		// Filter out .local domains
		if strings.HasSuffix(n, ".local") {
			continue
		}

		filteredDNSNames = append(filteredDNSNames, n)
	}

	if len(filteredDNSNames) == 0 {
		return "", errors.New("could not determine cluster endpoint")
	}

	selectedName := filteredDNSNames[0]

	// Find the port the remote kubernetes API server is running on
	// by inspecting kubernetes.default.svc service and finding
	// the target port it forwards traffic to
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return "", err
	}

	kubernetesService, err := clientset.CoreV1().Services("default").Get(context.TODO(), "kubernetes", metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	var port string
	for _, p := range kubernetesService.Spec.Ports {
		if p.Name == "https" {
			port = p.TargetPort.String()
		}
	}

	if port != "" {
		selectedName += ":" + port
	}

	return selectedName, nil
}

// GetSecret returns a K8s secret object with the specified name from the current namespace if it exists
func (k *Kubernetes) GetSecret(secret string) (string, error) {
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return "", err
	}
	return k.SecretReader.Get(secret, clientset)
}
