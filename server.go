package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/square/go-jose/jwt"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

func TokenAuth(data *Data) gin.HandlerFunc {
	return func(c *gin.Context) {
		authuser, _, _ := c.Request.BasicAuth()

		tokensk := ""
		if strings.HasPrefix(authuser, "sk_") {
			tokensk = strings.Replace(authuser, "sk_", "", -1)
		}

		authorization := c.Request.Header.Get("Authorization")
		if strings.HasPrefix(authorization, "Bearer sk_") {
			tokensk = strings.Replace(authorization, "Bearer sk_", "", -1)
		}

		if len(tokensk) != IDLength+SecretKeyLength {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			fmt.Println("Secret key length invalid")
			return
		}

		fmt.Println("sk", tokensk)

		id := tokensk[0:IDLength]

		fmt.Println("id", id)
		token, err := data.GetToken("tk_"+id, true)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			fmt.Println(err)
			return
		}

		secret := tokensk[IDLength : IDLength+SecretKeyLength]
		if err := bcrypt.CompareHashAndPassword(token.HashedSecret, []byte(secret)); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if time.Now().After(time.Unix(token.Expires, 0)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "expired token"})
			return
		}

		user, err := data.GetUser(token.User)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no user"})
			return
		}

		c.Set("user", user.ID)
		c.Set("token", token.ID)

		c.Next()
	}
}

func ProxyHandler(data *Data, kubernetes *Kubernetes) gin.HandlerFunc {
	remote, err := url.Parse(kubernetes.Config.Host)
	if err != nil {
		fmt.Println(err)
	}

	ca, err := ioutil.ReadFile(kubernetes.Config.TLSClientConfig.CAFile)
	if err != nil {
		fmt.Println(err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(ca)

	// TODO(jmorganca): use roundtripper? kubernetes.Config.WrapTransport
	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}
	return func(c *gin.Context) {
		userID := c.GetString("user")
		user, err := data.GetUser(userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
			return
		}

		c.Request.Header.Del("Authorization")
		c.Request.Header.Set("Impersonate-User", user.Email)
		c.Request.Header.Add("Authorization", "Bearer "+string(kubernetes.Config.BearerToken))

		http.StripPrefix("/v1/proxy", proxy).ServeHTTP(c.Writer, c.Request)
	}
}

func UpdateKubernetesClusterRoleBindings(data *Data, kubernetes *Kubernetes) error {
	if data == nil {
		return errors.New("data cannot be nil")

	}
	if kubernetes == nil {
		return errors.New("data cannot be nil")
	}

	users, err := data.ListUsers()
	if err != nil {
		return err
	}

	roleBindings := []RoleBinding{}
	for _, user := range users {
		roleBindings = append(roleBindings, RoleBinding{User: user.Email, Role: user.Permission})
	}

	return kubernetes.UpdateRoleBindings(roleBindings)
}

func ValidPermission(permission string) bool {
	for _, p := range Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func addRoutes(router *gin.Engine, data *Data, kubernetes *Kubernetes, config *ServerConfig, auth bool) error {
	router.GET("/v1/providers", func(c *gin.Context) {
		// TODO: define this better
		c.JSON(http.StatusOK, gin.H{
			"okta": gin.H{
				"domain":    config.Providers.Okta.Domain,
				"client-id": config.Providers.Okta.ClientID,
			},
		})
	})

	router.POST("/v1/tokens", func(c *gin.Context) {
		type Params struct {
			Code string `form:"code"`
			User string `form:"user"`
		}

		var params Params
		if err := c.ShouldBind(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		targetUser := ""

		// If we're using an OIDC code, verify identity and provide token
		if params.Code != "" {
			ctx := context.Background()
			conf := &oauth2.Config{
				ClientID:     config.Providers.Okta.ClientID,
				ClientSecret: config.Providers.Okta.ClientSecret,
				RedirectURL:  "http://localhost:8301",
				Scopes:       []string{"openid", "email"},
				Endpoint: oauth2.Endpoint{
					TokenURL: "https://dev-02708987.okta.com/oauth2/v1/token",
					AuthURL:  "https://dev-02708987.okta.com/oauth2/v1/authorize",
				},
			}

			exchanged, err := conf.Exchange(ctx, params.Code)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
				return
			}

			raw := exchanged.Extra("id_token").(string)
			tok, err := jwt.ParseSigned(raw)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "could not verify user with identity provider"})
				return
			}

			fmt.Println(tok)

			out := make(map[string]interface{})

			// TODO: verify?
			tok.UnsafeClaimsWithoutVerification(&out)

			email := out["email"].(string)

			user, err := data.FindUser(email)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "user does not exist"})
				return
			}

			targetUser = user.ID
		} else {
			if auth {
				// TODO: refactor this
				TokenAuth(data)(c)
				if c.GetString("user") == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"})
					return
				}
			}
			targetUser = params.User
			if targetUser == "" {
				targetUser = c.GetString("user")
				data.DeleteToken(c.GetString("token"))
			}
		}

		created, sk, err := data.CreateToken(targetUser)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		token, err := data.GetToken(created.ID, true)
		fmt.Println("retrieved", token, err)

		fmt.Println("Created token for user", targetUser, created)

		c.JSON(http.StatusCreated, gin.H{
			"token":        created,
			"secret_token": sk,
		})
	})

	if auth {
		router.Use(TokenAuth(data))
	}

	router.GET("/v1/users", func(c *gin.Context) {
		users, err := data.ListUsers()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"object": "list", "url": "/v1/users", "has_more": false, "data": users})
	})

	router.POST("/v1/users", func(c *gin.Context) {
		type binds struct {
			Email      string `form:"email" binding:"required"`
			Permission string `form:"permission"`
		}

		var form binds
		if err := c.Bind(&form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if form.Permission != "" && !ValidPermission(form.Permission) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission"})
			return
		}

		user := &User{
			Email:      form.Email,
			Providers:  []string{"token"},
			Permission: form.Permission,
		}

		err := data.PutUser(user)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := UpdateKubernetesClusterRoleBindings(data, kubernetes); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, user)
	})

	router.DELETE("/v1/users/:id", func(c *gin.Context) {
		type binds struct {
			ID string `uri:"id" binding:"required"`
		}

		var params binds
		if err := c.BindUri(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if _, err := data.GetUser(params.ID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user does not exist"})
			return
		}

		if err := data.DeleteUser(params.ID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := UpdateKubernetesClusterRoleBindings(data, kubernetes); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Remove user from view ClusterRoleBinding
		c.JSON(http.StatusOK, gin.H{"object": "user", "id": params.ID, "deleted": true})
	})

	if kubernetes != nil {
		proxyHandler := ProxyHandler(data, kubernetes)
		router.GET("/v1/proxy/*all", proxyHandler)
		router.POST("/v1/proxy/*all", proxyHandler)
		router.PUT("/v1/proxy/*all", proxyHandler)
		router.PATCH("/v1/proxy/*all", proxyHandler)
		router.DELETE("/v1/proxy/*all", proxyHandler)
	}

	return nil
}

type ServerOptions struct {
	DBPath     string
	ConfigPath string
	TLSCache   string
}

type OktaConfig struct {
	Domain       string `yaml:"domain" json:"domain"`
	ClientID     string `yaml:"client-id" json:"client-id"`
	ClientSecret string `yaml:"client-secret"` // TODO(jmorganca): move this to a secret
	ApiToken     string `yaml:"api-token"`     // TODO(jmorganca): move this to a secret
}

type ServerConfig struct {
	Providers struct {
		Okta OktaConfig `yaml:"okta" json:"okta"`
	}
	Permissions []struct {
		User       string
		Group      string
		Permission string
	}
}

func loadConfig(path string) (*ServerConfig, error) {
	contents, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return &ServerConfig{}, nil
	}

	if err != nil {
		return nil, err
	}

	var config ServerConfig
	err = yaml.Unmarshal([]byte(contents), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func ServerRun(options *ServerOptions) error {
	if options.DBPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		options.DBPath = filepath.Join(homeDir, ".infra")
	}

	data, err := NewData(options.DBPath)
	if err != nil {
		return err
	}

	defer data.Close()

	config, err := loadConfig(options.ConfigPath)
	if err != nil {
		return err
	}

	kubernetes, err := NewKubernetes()
	if err != nil {
		fmt.Println("warning: no kubernetes cluster detected.")
	}

	okta := &Okta{
		Domain:     config.Providers.Okta.Domain,
		ClientID:   config.Providers.Okta.ClientID,
		ApiToken:   config.Providers.Okta.ApiToken,
		Data:       data,
		Kubernetes: kubernetes,
	}

	// TODO (jmorganca): sync should be outside of Okta - with other providers and Kubernetes
	okta.Start()

	gin.SetMode(gin.ReleaseMode)

	unixRouter := gin.New()
	if err = addRoutes(unixRouter, data, kubernetes, config, false); err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Join(homeDir, ".infra"), os.ModePerm); err != nil {
		return err
	}

	os.Remove(filepath.Join(homeDir, ".infra", "infra.sock"))
	go func() {
		if err := unixRouter.RunUnix(filepath.Join(homeDir, ".infra", "infra.sock")); err != nil {
			log.Fatal(err)
		}
	}()

	router := gin.New()
	if err = addRoutes(router, data, kubernetes, config, true); err != nil {
		return err
	}

	m := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
	}

	fmt.Println("Using certificate cache", options.TLSCache)

	if options.TLSCache != "" {
		m.Cache = autocert.DirCache(options.TLSCache)
	}

	tlsServer := &http.Server{
		Addr:      ":8443",
		TLSConfig: m.TLSConfig(),
		Handler:   router,
	}

	return tlsServer.ListenAndServeTLS("", "")
}
