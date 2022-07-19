package kubernetes

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/infrahq/secrets"
	"github.com/jessevdk/go-flags"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"

	"github.com/infrahq/infra/internal/logging"
)

const (
	podLabelsFilePath = "/etc/podinfo/labels"
	namespaceFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	caFilePath        = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

type Kubernetes struct {
	Config       *rest.Config
	SecretReader secrets.SecretStorage
}

func NewKubernetes() (*Kubernetes, error) {
	k := &Kubernetes{}

	config, err := rest.InClusterConfig()
	if err != nil {
		return k, err
	}

	k.Config = config

	namespace, err := Namespace()
	if err != nil {
		return k, err
	}

	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return nil, err
	}

	k.SecretReader = secrets.NewKubernetesSecretProvider(clientset, namespace)

	return k, err
}

// namespaceRole is used as a tuple to pair namespaces and grants as a map key
type ClusterRoleNamespace struct {
	ClusterRole string
	Namespace   string
}

// UpdateClusterRoleBindings generates ClusterRoleBindings for GrantMappings
func (k *Kubernetes) UpdateClusterRoleBindings(subjects map[string][]rbacv1.Subject) error {
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return err
	}

	// store which cluster-roles currently exist locally
	validClusterRoles := make(map[string]bool)

	crs, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, cr := range crs.Items {
		validClusterRoles[cr.Name] = true
	}

	crbs := []*rbacv1.ClusterRoleBinding{}

	for cr, subjs := range subjects {
		if !validClusterRoles[cr] {
			logging.Warnf("cluster role binding %s skipped, it does not exist", cr)
			continue
		}

		crbs = append(crbs, &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("infra:%s", cr),
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "infra",
				},
			},
			Subjects: subjs,
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     cr,
			},
		})
	}

	existingInfraCrbs, err := clientset.RbacV1().ClusterRoleBindings().List(context.Background(), metav1.ListOptions{LabelSelector: "app.kubernetes.io/managed-by=infra"})
	if err != nil {
		return err
	}

	toDelete := make(map[string]bool)
	for _, existingCrb := range existingInfraCrbs.Items {
		toDelete[existingCrb.Name] = true
	}

	// Create or update CRBs for users
	for _, crb := range crbs {
		_, err = clientset.RbacV1().ClusterRoleBindings().Update(context.Background(), crb, metav1.UpdateOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.Background(), crb, metav1.CreateOptions{})
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		delete(toDelete, crb.Name)
	}

	for name := range toDelete {
		err := clientset.RbacV1().ClusterRoleBindings().Delete(context.Background(), name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (k *Kubernetes) UpdateRoleBindings(subjects map[ClusterRoleNamespace][]rbacv1.Subject) error {
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return err
	}

	// store which cluster-roles currently exist locally
	validClusterRoles := make(map[string]bool)

	crs, err := clientset.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, cr := range crs.Items {
		validClusterRoles[cr.Name] = true
	}

	// create the namespaced role bindings for all the users of each of the role assignments
	rbs := []*rbacv1.RoleBinding{}

	for crn, subjs := range subjects {
		if !validClusterRoles[crn.ClusterRole] {
			logging.Warnf("cluster role binding %s skipped, it does not exist", crn.ClusterRole)
			continue
		}

		rbs = append(rbs, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("infra:%s", crn.ClusterRole),
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "infra",
				},
				Namespace: crn.Namespace,
			},
			Subjects: subjs,
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     crn.ClusterRole,
			},
		})
	}

	existingInfraRbs, err := clientset.RbacV1().RoleBindings("").List(context.TODO(), metav1.ListOptions{LabelSelector: "app.kubernetes.io/managed-by=infra"})
	if err != nil {
		return err
	}

	type rbIdentifier struct {
		namespace string
		name      string
	}

	toDelete := make(map[rbIdentifier]rbacv1.RoleBinding)

	for _, existingRb := range existingInfraRbs.Items {
		rbID := rbIdentifier{
			namespace: existingRb.Namespace,
			name:      existingRb.Name,
		}
		toDelete[rbID] = existingRb
	}

	// Create or update RoleBindings for users/groups
	for _, rb := range rbs {
		_, err = clientset.RbacV1().RoleBindings(rb.Namespace).Update(context.TODO(), rb, metav1.UpdateOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				_, err = clientset.RbacV1().RoleBindings(rb.Namespace).Create(context.TODO(), rb, metav1.CreateOptions{})
				if err != nil {
					if k8sErrors.IsNotFound(err) {
						// the namespace does not exist
						// we can proceed in this case, the role mapping is just not applicable to this cluster
						logging.Warnf("skipping unapplicable namespace for this cluster: %s %s", rb.Namespace, err.Error())
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

	// Delete any Role-kind RoleBindings managed by infra that aren't in the config
	// Do not need to worry about deleted namespaces as they will also delete all their resources
	for _, td := range toDelete {
		err := clientset.RbacV1().RoleBindings(td.Namespace).Delete(context.TODO(), td.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (k *Kubernetes) Namespaces() ([]string, error) {
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return nil, err
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	results := make([]string, len(namespaces.Items))
	for i, n := range namespaces.Items {
		results[i] = n.Name
	}

	return results, nil
}

func ec2InstanceMetadata(metadata string) ([]byte, error) {
	req, err	:= http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("http://169.254.169.254/latest/meta-data/%s", metadata), nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("received non-OK code from metadata service")
	}

	return ioutil.ReadAll(res.Body)
}

func ec2Sign(key, message []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

func ec2SigningKey(secretAccessKey, date, region, service string) []byte {
	key := []byte(fmt.Sprintf("AWS4%s", secretAccessKey))
	key = ec2Sign(key, []byte(date))
	key = ec2Sign(key, []byte(region))
	key = ec2Sign(key, []byte(service))
	return ec2Sign(key, []byte("aws4_request"))
}

func (k *Kubernetes) ec2ClusterName() (string, error) {
	// gather metadata needed to construct the request
	instanceID, err := ec2InstanceMetadata("/instance-id")
	if err != nil {
		return "", err
	}

	region, err := ec2InstanceMetadata("/placement/region")
	if err != nil {
		return "", err
	}

	roleName, err := ec2InstanceMetadata("/iam/security-credentials")
	if err != nil {
		return "", err
	}

	bts, err := ec2InstanceMetadata(fmt.Sprintf("/iam/security-credentials/%s", roleName))
	if err != nil {
		return "", err
	}

	var credentials struct {
		AccessKeyID string `json:"AccessKeyId"`
		SecretAccessKey string `json:"SecretAccessKey"`
		SecurityToken string `json:"Token"`
	}

	if err := json.Unmarshal(bts, &credentials); err != nil {
		return "", errors.New("unmarshal security credentials")
	}

	ts := time.Now().UTC()
	datetime := ts.Format("20060102T150405Z")
	date := ts.Format("20060102")
	host := fmt.Sprintf("ec2.%s.amazonaws.com", region)

	// construct the request: https://docs.aws.amazon.com/general/latest/gr/sigv4_signing.html
	credentialScope := strings.Join([]string{date, string(region), "ec2", "aws4_request",}, "/")

	canonicalQueryString := strings.Join([]string{
		"Action=DescribeInstances",
		fmt.Sprintf("InstanceId.1=%s", instanceID),
		"Version=2016-11-15",
		"X-Amz-Algorithm=AWS4-HMAC-SHA256",
		fmt.Sprintf("X-Amz-Credential=%s", url.QueryEscape(strings.Join([]string{credentials.AccessKeyID, credentialScope}, "/"))),
		fmt.Sprintf("X-Amz-Date=%s", datetime),
		fmt.Sprintf("X-Amz-Security-Token=%s", url.QueryEscape(credentials.SecurityToken)),
		fmt.Sprintf("X-Amz-SignedHeaders=host"),
	}, "&")

	// DescribeInstances has no payload
	payloadChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte("")))

	canonicalRequest := strings.Join([]string{
		http.MethodGet, "/", canonicalQueryString, fmt.Sprintf("host:%s\n", host), "host", payloadChecksum,
	}, "\n")

	requestChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte(canonicalRequest)))

	message := strings.Join([]string{"AWS4-HMAC-SHA256", datetime, credentialScope, requestChecksum}, "\n")

	signingKey := ec2SigningKey(credentials.SecretAccessKey, date, string(region), "ec2")
	signature := ec2Sign(signingKey, []byte(message))

	fullRequest := fmt.Sprintf("https://%s?%s&X-Amz-Signature=%x", host, canonicalQueryString, string(signature))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fullRequest, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", errors.New("received non-OK code")
	}

	raw, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	type tag struct {
		Key string `xml:"key"`
		value string `xml:"value"`
	}

	var describeInstancesResponse struct {
		XMLName xml.Name `xml."DescribeInstancesResponse"`
		Tags []tag `xml:"reservationSet>item>instancesSet>item>tagSet>item"`
	}

	if err := xml.Unmarshal(raw, &describeInstancesResponse); err != nil {
		return "", err
	}

	for _, tag := range describeInstancesResponse.Tags {
		if strings.HasPrefix(tag.Key, "kubernetes.io/cluster/") {
			return strings.SplitN(tag.Key, "/", 3)[2], nil
		}
	}

	return "", errors.New("could not get aws cluster name")
}

func (k *Kubernetes) gkeClusterName() (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://169.254.169.254/computeMetadata/v1/instance/attributes/cluster-name", nil)
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://169.254.169.254/metadata/instance/compute/resourceGroupName?api-version=2017-08-01&format=text", nil)
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
		return "", fmt.Errorf("error while reading response from azure metadata endpoint: %w", err)
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

	pods := k8sAppPods.Items
	pods = append(pods, componentPods.Items...)

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

func (k *Kubernetes) Checksum() (string, error) {
	ca, err := CA()
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(ca)
	hash := h.Sum(nil)
	return hex.EncodeToString(hash), nil
}

func (k *Kubernetes) Name(chksm string) (string, error) {
	name := chksm[:12]

	// 169.254.169.254 is an address used by cloud platforms for instance metadata
	if _, err := net.DialTimeout("tcp", "169.254.169.254:80", 1*time.Second); err == nil {
		if name, err := k.ec2ClusterName(); err == nil {
			return name, nil
		}

		if name, err := k.gkeClusterName(); err == nil {
			return name, nil
		}

		if name, err := k.aksClusterName(); err == nil {
			return name, nil
		}
	}

	if name, err := k.kubeControllerManagerClusterName(); err == nil {
		return name, nil
	}

	// truncated checksum will be used as default name if one could not be found
	logging.Debugf("could not fetch cluster name, resorting to hashed cluster CA")

	return name, nil
}

func PodLabels() ([]string, error) {
	contents, err := ioutil.ReadFile(podLabelsFilePath)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(contents), "\n"), nil
}

// InstancePodLabels returns all pod labels with the prefix "app.kubernetes.io/instance"
func InstancePodLabels() ([]string, error) {
	podLabels, err := PodLabels()
	if err != nil {
		return nil, err
	}

	instanceLabels := []string{}
	for _, label := range podLabels {
		if strings.HasPrefix(label, "app.kubernetes.io/instance") {
			instanceLabels = append(instanceLabels, strings.ReplaceAll(label, "\"", ""))
			break
		}
	}

	return instanceLabels, nil
}

func Namespace() (string, error) {
	contents, err := ioutil.ReadFile(namespaceFilePath)
	if err != nil {
		return "", err
	}

	return string(contents), nil
}

func CA() ([]byte, error) {
	contents, err := ioutil.ReadFile(caFilePath)
	if err != nil {
		return nil, err
	}

	return contents, nil
}

// Find the first suitable Service, filtering on infrahq.com/component
func (k *Kubernetes) Service(component string, labels ...string) (*corev1.Service, error) {
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return nil, err
	}

	namespace, err := Namespace()
	if err != nil {
		return nil, err
	}

	selector := []string{
		fmt.Sprintf("app.infrahq.com/component=%s", component),
	}

	selector = append(selector, labels...)

	services, err := clientset.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: strings.Join(selector, ","),
	})
	if err != nil {
		return nil, err
	}

	if len(services.Items) == 0 {
		return nil, fmt.Errorf("no service found for component %s", component)
	}

	return &services.Items[0], nil
}

// Find a suitable Endpoint to use by inspecting Service objects
func (k *Kubernetes) Endpoint() (string, int, error) {
	labels, err := InstancePodLabels()
	if err != nil {
		return "", -1, err
	}

	service, err := k.Service("connector", labels...)
	if err != nil {
		return "", -1, err
	}

	var host string

	// nolint:exhaustive
	switch service.Spec.Type {
	case corev1.ServiceTypeClusterIP:
		host = service.Spec.ClusterIP
	case corev1.ServiceTypeNodePort:
		fallthrough
	case corev1.ServiceTypeLoadBalancer:
		if len(service.Status.LoadBalancer.Ingress) == 0 {
			return "", -1, fmt.Errorf("load balancer has no ingress objects")
		}

		ingress := service.Status.LoadBalancer.Ingress[0]

		host = ingress.Hostname
		if host == "" {
			host = ingress.IP
		}
	default:
		return "", -1, fmt.Errorf("unsupported service type")
	}

	if len(service.Spec.Ports) == 0 {
		return "", -1, fmt.Errorf("service has no ports")
	}

	return host, int(service.Spec.Ports[0].Port), nil
}

func (k *Kubernetes) IsServiceTypeClusterIP() (bool, error) {
	labels, err := InstancePodLabels()
	if err != nil {
		return false, err
	}
	service, err := k.Service("connector", labels...)
	if err != nil {
		return false, err
	}

	return service.Spec.Type == corev1.ServiceTypeClusterIP, nil
}

func (k *Kubernetes) ClusterRoles() ([]string, error) {
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return nil, err
	}

	rbacDefaults, err := clientset.RbacV1().ClusterRoles().List(context.Background(), metav1.ListOptions{
		LabelSelector: "kubernetes.io/bootstrapping=rbac-defaults",
	})
	if err != nil {
		return nil, err
	}

	infraRoles, err := clientset.RbacV1().ClusterRoles().List(context.Background(), metav1.ListOptions{
		LabelSelector: "app.infrahq.com/include-role=true",
	})
	if err != nil {
		return nil, err
	}

	results := make([]string, 0, len(rbacDefaults.Items)+len(infraRoles.Items))
	for _, n := range append(rbacDefaults.Items, infraRoles.Items...) {
		if strings.HasPrefix(n.Name, "system:") {
			continue
		}

		results = append(results, n.Name)
	}

	return results, nil
}
