package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/infrahq/infra/internal/data"
	"github.com/infrahq/infra/internal/kubernetes"
	"github.com/infrahq/infra/internal/util"
	"github.com/square/go-jose/jwt"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

func TokenAuth(db *data.Data) gin.HandlerFunc {
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

		if len(tokensk) != data.IDLength+data.SecretKeyLength {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			fmt.Println("Secret key length invalid")
			return
		}

		fmt.Println("sk", tokensk)

		id := tokensk[0:data.IDLength]

		fmt.Println("id", id)
		token, err := db.GetToken("tk_"+id, true)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			fmt.Println(err)
			return
		}

		secret := tokensk[data.IDLength : data.IDLength+data.SecretKeyLength]
		if err := bcrypt.CompareHashAndPassword(token.HashedSecret, []byte(secret)); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if time.Now().After(time.Unix(token.Expires, 0)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "expired token"})
			return
		}

		user, err := db.GetUser(token.User)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no user"})
			return
		}

		c.Set("user", user.ID)
		c.Set("token", token.ID)

		c.Next()
	}
}

func ProxyHandler(data *data.Data, kubernetes *kubernetes.Kubernetes) gin.HandlerFunc {
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

func addRoutes(router *gin.Engine, d *data.Data, kube *kubernetes.Kubernetes, config *ServerConfig, auth bool) error {
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

			user, err := d.FindUser(email)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "user does not exist"})
				return
			}

			targetUser = user.ID
		} else {
			if auth {
				// TODO: refactor this
				TokenAuth(d)(c)
				if c.GetString("user") == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"})
					return
				}
			}
			targetUser = params.User
			if targetUser == "" {
				targetUser = c.GetString("user")
				d.DeleteToken(c.GetString("token"))
			}
		}

		token := &data.Token{
			User: targetUser,
		}

		sk, err := d.PutToken(token)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		token.HashedSecret = []byte{}

		c.JSON(http.StatusCreated, gin.H{
			"token":        token,
			"secret_token": sk,
		})
	})

	if auth {
		router.Use(TokenAuth(d))
	}

	router.GET("/v1/users", func(c *gin.Context) {
		users, err := d.ListUsers()
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

		if form.Permission != "" && !util.ValidPermission(form.Permission) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission"})
			return
		}

		user := &data.User{
			Email:      form.Email,
			Providers:  []string{"token"},
			Permission: form.Permission,
		}

		err := d.PutUser(user)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := UpdatePermissions(d, kube); err != nil {
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

		if _, err := d.GetUser(params.ID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user does not exist"})
			return
		}

		if err := d.DeleteUser(params.ID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := UpdatePermissions(d, kube); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Remove user from view ClusterRoleBinding
		c.JSON(http.StatusOK, gin.H{"object": "user", "id": params.ID, "deleted": true})
	})

	if kube != nil {
		proxyHandler := ProxyHandler(d, kube)
		router.GET("/v1/proxy/*all", proxyHandler)
		router.POST("/v1/proxy/*all", proxyHandler)
		router.PUT("/v1/proxy/*all", proxyHandler)
		router.PATCH("/v1/proxy/*all", proxyHandler)
		router.DELETE("/v1/proxy/*all", proxyHandler)
	}

	return nil
}
