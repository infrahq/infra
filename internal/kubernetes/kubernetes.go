package kubernetes

import (
	"context"
	"crypto/sha256"
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
	"github.com/jessevdk/go-flags"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/infrahq/infra/internal/api"
	"github.com/infrahq/infra/internal/logging"
	"github.com/infrahq/infra/secrets"
)

const (
	NamespaceFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	CAFilePath        = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

type BindingID struct {
	name string
	namespace string
}

type Binding struct {
	role string
	namespace string
}

type Kubernetes struct {
	mu           sync.Mutex
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

	namespace, err := k.Namespace()
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

func (k *Kubernetes) updateBindings(bindings map[Binding][]rbacv1.Subject) error {
	ctx := context.Background()
	options := metav1.ListOptions{LabelSelector: "app.kubernetes.io/managed-by=infra"}

	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return err
	}

	rbs, err := clientset.RbacV1().RoleBindings("").List(ctx, options)
	if err != nil {
		return err
	}

	knownRBs := make(map[BindingID]bool)
	for _, rb := range rbs.Items {
		knownRBs[BindingID{name: rb.Name, namespace: rb.Namespace}] = true
	}

	crbs, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, options)
	if err != nil {
		return err
	}

	knownCRBs := make(map[BindingID]bool)
	for _, crb := range crbs.Items {
		knownCRBs[BindingID{name: crb.Name}] = true
	}

	for binding, subjects := range bindings {
		if !strings.HasPrefix(binding.role, "cluster-") && binding.namespace != "" {
			if err := k.updateRoleBindings(ctx, clientset, binding, subjects, knownRBs); err != nil {
				logging.S.Infof("update role binding: %w", err)
			}
			continue
		}

		if err := k.updateClusterRoleBindings(ctx, clientset, binding, subjects, knownCRBs); err != nil {
			logging.S.Infof("update cluster role binding: %w", err)
		}
	}

	for rb := range knownRBs {
		if err := clientset.RbacV1().RoleBindings(rb.namespace).Delete(ctx, rb.name, metav1.DeleteOptions{}); err != nil {
			logging.S.Infof("delete role binding: %w", err)
			continue
		}
	}

	for crb := range knownCRBs {
		if err := clientset.RbacV1().ClusterRoleBindings().Delete(ctx, crb.name, metav1.DeleteOptions{}); err != nil {
			logging.S.Infof("delete cluster role binding: %w", err)
			continue
		}
	}

	return nil
}

func (k *Kubernetes) updateRoleBindings(ctx context.Context, clientset *kubernetes.Clientset, binding Binding, subjects []rbacv1.Subject, known map[BindingID]bool) error {
	name := fmt.Sprintf("infra:%s", binding.role)
	id := BindingID{name: name, namespace: binding.namespace}

	rb := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "infra",
			},
			Namespace: binding.namespace,
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind: "Role",
			Name: binding.role,
		},
	}

	var err error

	if _, ok := known[id]; !ok {
		_, err = clientset.RbacV1().RoleBindings(binding.namespace).Create(ctx, &rb, metav1.CreateOptions{})
	} else {
		_, err = clientset.RbacV1().RoleBindings(binding.namespace).Update(ctx, &rb, metav1.UpdateOptions{})
	}

	if err != nil {
		return err
	}

	delete(known, id)

	return nil
}

func (k *Kubernetes) updateClusterRoleBindings(ctx context.Context, clientset *kubernetes.Clientset, binding Binding, subjects []rbacv1.Subject, known map[BindingID]bool) error {
	name := fmt.Sprintf("infra:%s", binding.role)
	id := BindingID{name: name}

	crb := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "infra",
			},
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind: "ClusterRole",
			Name: binding.role,
		},
	}

	var err error

	if _, ok := known[id]; !ok {
		_, err = clientset.RbacV1().ClusterRoleBindings().Create(ctx, &crb, metav1.CreateOptions{})
	} else {
		_, err = clientset.RbacV1().ClusterRoleBindings().Update(ctx, &crb, metav1.UpdateOptions{})
	}

	if err != nil {
		return err
	}

	delete(known, id)

	return nil
}

// UpdateRoles converts infra grants to role-bindings in the current cluster
func (k *Kubernetes) UpdateRoles(grants []api.Grant) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	bindings := make(map[Binding][]rbacv1.Subject)

	for _, g := range grants {
		if g.Resource.Kind != "kubernetes" {
			continue
		}

		binding := Binding{
			role: g.Role,
			namespace: g.Resource.Path,
		}

		for _, u := range g.GetUsers() {
			bindings[binding] = append(bindings[binding], rbacv1.Subject{
				APIGroup: "rbac.authorization.k8s.io",
				Kind: "User",
				Name: u.Email,
			})
		}

		for _, g := range g.GetGroups() {
			bindings[binding] = append(bindings[binding], rbacv1.Subject{
				APIGroup: "rbac.authorization.k8s.io",
				Kind: "Group",
				Name: g.Name,
			})
		}
	}

	if err := k.updateBindings(bindings); err != nil {
		return fmt.Errorf("update bindings: %w", err)
	}

	return nil
}

func (k *Kubernetes) ec2ClusterName() (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://169.254.169.254/latest/dynamic/instance-identity/document", nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

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

	awsSess, err := session.NewSession(&aws.Config{Region: aws.String(identity.Region)})
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

func (k *Kubernetes) Name() (string, string, error) {
	ca, err := k.CA()
	if err != nil {
		return "", "", err
	}

	h := sha256.New()
	h.Write(ca)
	hash := h.Sum(nil)
	chksm := hex.EncodeToString(hash)

	name, err := k.ec2ClusterName()
	if err == nil {
		return name, chksm, nil
	}

	logging.S.Debugf("could not fetch ec2 cluster name: %s", err.Error())

	name, err = k.gkeClusterName()
	if err == nil {
		return name, chksm, nil
	}

	logging.S.Debugf("could not fetch gke cluster name: %s", err.Error())

	name, err = k.aksClusterName()
	if err == nil {
		return name, chksm, nil
	}

	logging.S.Debugf("could not fetch aks cluster name: %s", err.Error())

	name, err = k.kubeControllerManagerClusterName()
	if err == nil {
		return name, chksm, nil
	}

	logging.S.Debugf("could not fetch kube-controller-manager cluster name: %s", err.Error())

	logging.L.Debug("could not fetch cluster name, resorting to hashed cluster CA")

	// truncated checksum will be used as default name if one could not be found
	return chksm[:12], chksm, nil
}

func (k *Kubernetes) Namespace() (string, error) {
	contents, err := ioutil.ReadFile(NamespaceFilePath)
	if err != nil {
		return "", err
	}

	return string(contents), nil
}

func (k *Kubernetes) CA() ([]byte, error) {
	contents, err := ioutil.ReadFile(CAFilePath)
	if err != nil {
		return nil, err
	}

	return contents, nil
}

// Find the first suitable Service, filtering on infrahq.com/component
func (k *Kubernetes) Service(component string) (*corev1.Service, error) {
	clientset, err := kubernetes.NewForConfig(k.Config)
	if err != nil {
		return nil, err
	}

	namespace, err := k.Namespace()
	if err != nil {
		return nil, err
	}

	services, err := clientset.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("infrahq.com/component=%s", component),
	})
	if err != nil {
		return nil, err
	}

	if len(services.Items) == 0 {
		return nil, fmt.Errorf("no service found for component %s", component)
	}

	return &services.Items[0], nil
}

// Find a suitable Endpoint to use by inspecting the engine's Service objects
func (k *Kubernetes) Endpoint() (string, int, error) {
	service, err := k.Service("engine")
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
