package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/docker/go-units"
	"github.com/gin-gonic/gin"
	"github.com/olekukonko/tablewriter"
	"github.com/rs/xid"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type User struct {
	ID             string `gorm:"primaryKey" json:"id"`
	Username       string `json:"username"`
	HashedPassword []byte `json:"-"`
	Created        int    `gorm:"autoCreateTime" json:"created"`
	Updated        int    `gorm:"autoUpdateTime" json:"updated"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = "usr_" + xid.New().String()
	return nil
}

// Run runs the infra server
func Server() error {
	db, err := gorm.Open(sqlite.Open("infra.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&User{})

	var admin User
	if err = db.Where("username = ?", "admin").First(&admin).Error; err != nil {
		password, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			panic("could not hash password")
		}
		db.Create(&User{Username: "admin", HashedPassword: password})
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.GET("/v1/users", func(c *gin.Context) {
		var users []User
		db.Find(&users)
		c.JSON(http.StatusOK, gin.H{"object": "list", "url": "/v1/users", "has_more": false, "data": users})
	})

	router.GET("/v1/users/:id", func(c *gin.Context) {
		type binds struct {
			ID string `uri:"id" binding:"required"`
		}

		var params binds
		if err := c.BindUri(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user User
		if err := db.Where("id = ?", params.ID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, user)
	})

	router.POST("/v1/users", func(c *gin.Context) {
		type binds struct {
			Username string `form:"username" binding:"required"`
			Password string `form:"password" binding:"required"`
		}

		var form binds
		if err := c.Bind(&form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user := &User{Username: form.Username, HashedPassword: hashedPassword}
		if err = db.Create(&user).Error; err != nil {
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

		result := db.Delete(&User{ID: params.ID})
		if result.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
			return
		}

		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"object": "user", "id": params.ID, "deleted": true})
	})

	// Generate credentials for user
	router.POST("/v1/login", func(c *gin.Context) {
		type binding struct {
			Username string `form:"username" binding:"required"`
			Password string `form:"password" binding:"required"`
		}

		var params binding
		if err := c.ShouldBind(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user User
		if err := db.First(&user, "username = ?", params.Username).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "incorrect credentials"})
			return
		}

		if err = bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(params.Password)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "incorrect credentials"})
			return
		}

		// Creating Access Token
		claims := jwt.MapClaims{}
		claims["user"] = user.Username
		claims["exp"] = time.Now().Add(time.Minute * 5).Unix()
		unsigned := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token, err := unsigned.SignedString([]byte("secret")) // TODO: sign with same keypair certificate as certs
		if err != nil {
			panic(err)
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
		})
	})

	remote, err := url.Parse("https://kubernetes.default")
	if err != nil {
		log.Println("could not parse kubernetes endpoint")
	}

	// Load ca
	ca, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		log.Println("could not open cluster ca")
	}
	satoken, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		log.Println("could not open service account token")
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(ca)

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}

	stripProxy := http.StripPrefix("/v1/proxy", proxy)
	proxyHandler := func(c *gin.Context) {
		authorization := c.Request.Header.Get("Authorization")

		claims := jwt.MapClaims{}
		jwt.ParseWithClaims(strings.Split(authorization, " ")[1], claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})

		fmt.Printf("%+v\n", claims)

		c.Request.Header.Set("Impersonate-User", claims["user"].(string))
		c.Request.Header.Del("Authorization")
		c.Request.Header.Add("Authorization", "Bearer "+string(satoken))
		stripProxy.ServeHTTP(c.Writer, c.Request)
	}

	// Access proxy endpoints
	router.GET("/v1/proxy/*all", proxyHandler)
	router.POST("/v1/proxy/*all", proxyHandler)
	router.PUT("/v1/proxy/*all", proxyHandler)
	router.PATCH("/v1/proxy/*all", proxyHandler)
	router.DELETE("/v1/proxy/*all", proxyHandler)

	// // SCIM endpoints
	// router.GET("/scim/v2/Users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// 	fmt.Printf("%+v\n", r)
	// })
	// router.POST("/scim/v2/Users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// 	fmt.Printf("%+v\n", r)
	// })
	// router.PUT("/scim/v2/Users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// 	fmt.Printf("%+v\n", r)
	// })
	// router.PATCH("/scim/v2/Users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// 	fmt.Printf("%+v\n", r)
	// })

	fmt.Printf("Listening on port %v\n", 3001)
	router.Run(":3001")

	return nil
}

func PrintTable(header []string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(header)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	table.AppendBulk(data)
	table.Render()
}

func main() {
	app := &cli.App{
		Usage: "Manage infrastructure identity & access",
		Commands: []*cli.Command{
			{
				Name:  "user",
				Usage: "Manage users",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "add a new user",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:     "username",
								Aliases:  []string{"u"},
								Required: true,
							},
							&cli.StringFlag{
								Name:     "password",
								Aliases:  []string{"p"},
								Required: true,
							},
						},
						Action: func(c *cli.Context) error {
							form := url.Values{}
							form.Add("username", c.String("username"))
							form.Add("password", c.String("password"))

							_, err := http.PostForm("http://localhost:3001/v1/users", url.Values{
								"username": {c.String("username")},
								"password": {c.String("password")},
							})
							if err != nil {
								panic("http request failed")
							}

							fmt.Println("User added")

							return nil
						},
					},
					{
						Name:    "list",
						Usage:   "list users",
						Aliases: []string{"ls"},
						Action: func(c *cli.Context) error {
							res, err := http.Get("http://localhost:3001/v1/users")
							if err != nil {
								panic("http request failed")
							}

							type response struct {
								Data []struct {
									ID       string `json:"id"`
									Username string `json:"username"`
									Created  int64  `json:"created"`
									Updated  int64  `json:"updated"`
								}
							}

							var decoded response
							if err = json.NewDecoder(res.Body).Decode(&decoded); err != nil {
								panic(err)
							}

							rows := [][]string{}
							for _, user := range decoded.Data {
								createdAt := time.Unix(user.Created, 0)

								rows = append(rows, []string{user.Username, user.ID, units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago"})
							}

							PrintTable([]string{"USERNAME", "ID", "CREATED"}, rows)

							return nil
						},
					},
					{
						Name:  "delete",
						Usage: "delete a user",
						Action: func(c *cli.Context) error {
							req, err := http.NewRequest(http.MethodDelete, "http://localhost:3001/v1/users/"+c.Args().Get(0), nil)
							if err != nil {
								panic(err)
							}

							client := &http.Client{}
							res, err := client.Do(req)
							if err != nil {
								panic(err)
							}
							defer res.Body.Close()

							type response struct {
								ID      string `json:"id"`
								Deleted bool   `json:"deleted"`
								Error   string `json:"error"`
							}

							var decoded response
							if err = json.NewDecoder(res.Body).Decode(&decoded); err != nil {
								panic(err)
							}

							if decoded.Deleted {
								fmt.Println("User deleted")
							} else if len(decoded.Error) > 0 {
								fmt.Println("Could not delete user: " + decoded.Error)
							} else {
								fmt.Println("Could not delete user")
							}

							return nil
						},
					},
				},
			},
			{
				Name:   "users",
				Hidden: true,
				Action: func(c *cli.Context) error {
					c.App.Run([]string{c.App.Name, "user", "ls"})
					return nil
				},
			},
			{
				Name:  "login",
				Usage: "Login to an Infra Engine",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "username",
						Aliases:  []string{"u"},
						Required: true,
					},
					&cli.StringFlag{
						Name:     "password",
						Aliases:  []string{"p"},
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					form := url.Values{}
					form.Add("username", c.String("username"))
					form.Add("password", c.String("password"))

					res, err := http.Post("http://localhost:3001/v1/login", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
					if err != nil {
						panic("http request failed")
					}

					type tokenResponse struct {
						Token string `json:"token"`
					}

					var tr tokenResponse
					if err = json.NewDecoder(res.Body).Decode(&tr); err != nil {
						panic(err)
					}

					fmt.Printf("%+v\n", tr.Token)

					loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
					// if you want to change the loading rules (which files in which order), you can do so here
					configOverrides := &clientcmd.ConfigOverrides{}
					// if you want to change override values or bind them to flags, there are methods to help you
					kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
					_, err = kubeConfig.RawConfig()

					// Create kubeconfig
					config := clientcmdapi.NewConfig()
					config.Clusters["infra"] = &clientcmdapi.Cluster{
						Server: "http://localhost:3001/v1/proxy",
					}
					config.AuthInfos["infra"] = &clientcmdapi.AuthInfo{
						Token: tr.Token,
					}
					config.Contexts["infra"] = &clientcmdapi.Context{
						Cluster:  "infra",
						AuthInfo: "infra",
					}
					config.CurrentContext = "infra"
					err = clientcmd.WriteToFile(*config, "config.yaml")
					fmt.Printf("%+v\n", config)
					fmt.Println("Kubeconfig updated")

					// Insert into kubeconfig

					return nil
				},
			},
			{
				Name:  "logout",
				Usage: "Log out of an Infra Engine",
				Action: func(c *cli.Context) error {
					fmt.Println("NOT IMPLEMENTED")
					return nil
				},
			},
			{
				Name:  "server",
				Usage: "Start the Infra Engine",
				Action: func(c *cli.Context) error {
					Server()
					return nil
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
