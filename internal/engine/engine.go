package engine

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goware/urlx"
	"github.com/infrahq/infra/internal/server"
	"github.com/infrahq/infra/internal/timer"
	"github.com/jessevdk/go-flags"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type Options struct {
	Server        string
	Name          string
	SkipTLSVerify bool
	APIKey        string
}

type RoleBinding struct {
	User string
	Role string
}

type RegistrationInfo struct {
	CA              string
	ClusterEndpoint string
}

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

func (k *Kubernetes) UpdatePermissions(rbs []RoleBinding) error {
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

	// TODO (jmorganca): find and delete empty rolebindings
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

	fmt.Println("todelete")
	fmt.Println(toDelete)

	for _, td := range toDelete {
		err := clientset.RbacV1().ClusterRoleBindings().Delete(context.TODO(), td.Name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

type jwkCache struct {
	mu          sync.Mutex
	key         jose.JSONWebKey
	lastChecked time.Time
}

var jwkCacheRefresh = 5 * time.Minute

var cache jwkCache

func (j *jwkCache) Get(client *http.Client, url string) (key jose.JSONWebKey, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.lastChecked != (time.Time{}) && time.Now().Before(j.lastChecked.Add(jwkCacheRefresh)) {
		key = j.key
		return
	}

	res, err := client.Get(url)
	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	var response struct {
		Keys []jose.JSONWebKey `json:"keys"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return
	}

	if len(response.Keys) < 1 {
		err = errors.New("no jwks provided by server")
		return
	}

	j.key = response.Keys[0]
	j.lastChecked = time.Now()

	return j.key, nil
}

func JWTMiddleware(client *http.Client, base string) func(c *gin.Context) {
	return func(c *gin.Context) {
		// Check bearer header
		authorization := c.Request.Header.Get("X-Infra-Authorization")
		raw := strings.Replace(authorization, "Bearer ", "", -1)
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		tok, err := jwt.ParseSigned(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		key, err := cache.Get(client, base+"/.well-known/jwks.json")
		if err != nil {
			fmt.Println(err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		out := make(map[string]interface{})
		cl := jwt.Claims{}
		if err := tok.Claims(key, &cl, &out); err != nil {
			fmt.Println("could not verify token", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		err = cl.Validate(jwt.Expected{
			Issuer: "infra",
			Time:   time.Now(),
		})
		switch {
		case errors.Is(err, jwt.ErrExpired):
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "expired"})
			return
		case err != nil:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		tok.UnsafeClaimsWithoutVerification(&out)

		email := out["email"].(string)

		c.Set("email", email)
		c.Next()
	}
}

func (k *Kubernetes) ProxyHandler() (handler gin.HandlerFunc, err error) {
	remote, err := url.Parse(k.config.Host)
	if err != nil {
		return
	}

	ca, err := ioutil.ReadFile(k.config.TLSClientConfig.CAFile)
	if err != nil {
		return
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(ca)
	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}

	return func(c *gin.Context) {
		email := c.GetString("email")
		c.Request.Header.Del("X-Infra-Authorization")
		c.Request.Header.Set("Impersonate-User", email)
		c.Request.Header.Add("Authorization", "Bearer "+string(k.config.BearerToken))
		http.StripPrefix("/proxy", proxy).ServeHTTP(c.Writer, c.Request)
	}, err
}

func fetchConfig(client *http.Client, base string, name string) ([]server.Grant, error) {
	params := url.Values{}
	params.Add("name", name)

	res, err := client.Get(base + "/v1/config?" + params.Encode())
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(data))

	er := &server.ErrorResponse{}
	err = json.Unmarshal(data, &er)
	if err != nil {
		return nil, err
	}

	if er.Error != "" {
		return nil, errors.New(er.Error)
	}

	if res.StatusCode >= http.StatusBadRequest {
		return nil, errors.New("received error status code: " + http.StatusText(res.StatusCode))
	}

	var response struct{ Data []server.Grant }
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (k *Kubernetes) CA() ([]byte, error) {
	contents, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return nil, err
	}
	return contents, nil
}

func (k *Kubernetes) ExecCat(pod string, namespace string, file string) (string, error) {
	clientset, err := kubernetes.NewForConfig(k.config)
	if err != nil {
		return "", err
	}

	cmd := []string{
		"/bin/cat",
		file,
	}
	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(pod).Namespace(namespace).SubResource("exec")
	req.VersionedParams(
		&v1.PodExecOptions{
			Command: cmd,
			Stdout:  true,
		},
		scheme.ParameterCodec,
	)

	exec, err := remotecommand.NewSPDYExecutor(k.config, "POST", req.URL())
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: io.Writer(&buf),
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Endpoint gets the cluster endpoint from within the pod
func (k *Kubernetes) Endpoint() (string, error) {
	// TODO (jmorganca): find and delete empty rolebindings
	// Create empty crbs for roles with no users
	clientset, err := kubernetes.NewForConfig(k.config)
	if err != nil {
		return "", err
	}

	var endpoint string

	// Get the full command line for kube-proxy pods
	// if --master is specified, use that
	// if --kubeconfig is specified, exec + cat to read that
	// if --config is specified, exec + cat the file the kubeconfig location, and read the kubeconfig

	pods1, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "k8s-app=kube-proxy",
	})
	if err != nil {
		return "", err
	}

	pods2, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "component=kube-proxy",
	})
	if err != nil {
		return "", err
	}

	pods := append(pods1.Items, pods2.Items...)

	if len(pods) == 0 {
		return "", errors.New("no kube-proxy pods to inspect")
	}

	pod := pods[0]

	var command []string
	for _, c := range pod.Spec.Containers {
		if c.Name == "kube-proxy" {
			command = c.Command
			break
		}
	}

	var opts struct {
		Master     string `long:"master"`
		Config     string `long:"config"`
		Kubeconfig string `long:"kubeconfig"`
	}

	p := flags.NewParser(&opts, flags.IgnoreUnknown)
	_, err = p.ParseArgs(command[1:])
	if err != nil {
		return "", err
	}

	switch {
	case opts.Master != "":
		endpoint = opts.Master
	case opts.Config != "":
		contents, err := k.ExecCat(pod.Name, "kube-system", opts.Config)
		if err != nil {
			return "", err
		}
		var raw map[string]interface{}
		err = yaml.Unmarshal([]byte(contents), &raw)
		if err != nil {
			return "", err
		}

		clientConnection, ok := raw["clientConnection"].(map[interface{}]interface{})
		if !ok {
			return "", errors.New("invalid kube-proxy config format")
		}
		kubeconfig, ok := clientConnection["kubeconfig"].(string)
		if !ok {
			return "", errors.New("invalid kube-proxy config format")
		}

		opts.Kubeconfig = kubeconfig
		fallthrough
	case opts.Kubeconfig != "":
		contents, err := k.ExecCat(pod.Name, "kube-system", opts.Kubeconfig)
		if err != nil {
			return "", err
		}

		cfg, err := clientcmd.NewClientConfigFromBytes([]byte(contents))
		if err != nil {
			return "", err
		}

		rc, err := cfg.ClientConfig()
		if err != nil {
			return "", err
		}

		endpoint = rc.Host
	}

	// Rewrite docker desktop
	if endpoint == "https://vm.docker.internal:6443" {
		endpoint = "https://kubernetes.docker.internal:6443"
	}

	if strings.HasSuffix(endpoint, ".internal.k8s.ondigitalocean.com") {
		endpoint = strings.Replace(endpoint, ".internal.k8s.ondigitalocean.com", ".k8s.ondigitalocean.com", -1)
	}

	// TODO (jmorganca): minikube

	// Could not get endpoint - must be passed via flag
	return endpoint, nil
}

type BearerTransport struct {
	Token     string
	Transport http.RoundTripper
}

func (b *BearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if b.Token != "" {
		req.Header.Set("Authorization", "Bearer "+b.Token)
	}
	return b.Transport.RoundTrip(req)
}

func Run(options Options) error {
	client := &http.Client{
		Transport: &BearerTransport{
			Token: options.APIKey,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: options.SkipTLSVerify,
				},
			},
		},
	}

	kubernetes, err := NewKubernetes()
	if err != nil {
		return err
	}

	uri, err := urlx.Parse(options.Server)
	if err != nil {
		return err
	}

	uri.Scheme = "https"

	timer := timer.Timer{}
	timer.Start(5, func() {
		fmt.Println("sync start")

		ca, err := kubernetes.CA()
		if err != nil {
			fmt.Println(err)
			return
		}

		endpoint, err := kubernetes.Endpoint()
		if err != nil {
			fmt.Println(err)
			return
		}

		form := url.Values{}
		form.Add("ca", string(ca))
		form.Add("endpoint", endpoint)
		form.Add("name", options.Name)

		res, err := client.PostForm(uri.String()+"/v1/register", form)
		if err != nil {
			fmt.Println(err)
			return
		}

		if res.StatusCode != http.StatusOK {
			fmt.Println("failed to register, code: ", res.StatusCode)
			return
		}

		// Fetch latest grants from server
		grants, err := fetchConfig(client, uri.String(), options.Name)
		if err != nil {
			fmt.Println(err)
			return
		}

		var rbs []RoleBinding
		for _, p := range grants {
			rbs = append(rbs, RoleBinding{User: p.User.Email, Role: p.Role.KubernetesRole})
		}

		err = kubernetes.UpdatePermissions(rbs)
		if err != nil {
			fmt.Println(err)
			return
		}
	})
	defer timer.Stop()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())

	proxyHandler, err := kubernetes.ProxyHandler()
	if err != nil {
		return err
	}

	router.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/proxy/*all", JWTMiddleware(client, uri.String()), proxyHandler)
	router.POST("/proxy/*all", JWTMiddleware(client, uri.String()), proxyHandler)
	router.PUT("/proxy/*all", JWTMiddleware(client, uri.String()), proxyHandler)
	router.PATCH("/proxy/*all", JWTMiddleware(client, uri.String()), proxyHandler)
	router.DELETE("/proxy/*all", JWTMiddleware(client, uri.String()), proxyHandler)

	fmt.Println("serving on port 80")
	return router.Run(":80")
}
