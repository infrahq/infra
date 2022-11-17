package connectors

import (
	"context"
	"net/http"
	"sync"

	"github.com/infrahq/infra/api"
	"github.com/infrahq/infra/internal"
)

type PluginConfig map[string]interface{}

func Run(ctx context.Context, options ServerOptions) error {
	// read connector config
	configuredPlugins := PluginConfig{}

	wg := sync.WaitGroup{}

	for name, pluginConfig := range configuredPlugins {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// get plugin init func
			pluginInitFn, ok := plugins[name]
			if !ok {
				// log error
				return
			}

			// initialize plugin
			plugin, err := pluginInitFn(pluginConfig)
			if err != nil {
				// log error
			}

			// run plugin
			err = plugin.Run(ctx)
			if err != nil {
				// log error
			}
		}()
	}

	wg.Wait()
	return nil
}

func getClientForDestination(ctx context.Context, options ServerOptions, dest api.Destination) *api.Client {
	return &api.Client{
		Name:      "connector",
		Version:   internal.Version,
		URL:       options.URL,
		AccessKey: options.AccessKey.String(),
		HTTP: http.Client{
			Transport: httpTransportFromOptions(options),
		},
		// Headers: http.Header{
		// 	"Infra-Destination": {checkSum}, // should be id
		// },
	}
}

// Problems:
// - destinations are created by the connectors and probably shouldn't be.
//	 - infra destination identifier needs to be coordinated between the backend and the connector

// 	con := connector{
// 		k8s:         k8s,
// 		client:      client,
// 		destination: destination,
// 		certCache:   certCache,
// 		options:     options,
// 	}
// 	group.Go(func() error {
// 		backOff := &backoff.ExponentialBackOff{
// 			InitialInterval:     2 * time.Second,
// 			MaxInterval:         time.Minute,
// 			RandomizationFactor: 0.2,
// 			Multiplier:          1.5,
// 		}
// 		waiter := repeat.NewWaiter(backOff)
// 		return syncGrantsToKubeBindings(ctx, con, waiter)
// 	})
// 	group.Go(func() error {
// 		// TODO: how long should this wait? Use exponential backoff on error?
// 		waiter := repeat.NewWaiter(backoff.NewConstantBackOff(30 * time.Second))
// 		for {
// 			if err := syncDestination(ctx, con); err != nil {
// 				logging.Errorf("failed to update destination in infra: %v", err)
// 			} else {
// 				waiter.Reset()
// 			}
// 			if err := waiter.Wait(ctx); err != nil {
// 				return err
// 			}
// 		}
// 	})

// 	ginutil.SetMode()
// 	router := gin.New()
// 	router.GET("/healthz", func(c *gin.Context) {
// 		c.Status(http.StatusOK)
// 	})

// 	kubeAPIAddr, err := urlx.Parse(k8s.Config.Host)
// 	if err != nil {
// 		return fmt.Errorf("parsing host config: %w", err)
// 	}

// 	certPool := x509.NewCertPool()
// 	if ok := certPool.AppendCertsFromPEM(k8s.Config.CAData); !ok {
// 		return errors.New("could not append CA to client cert bundle")
// 	}

// 	proxyTransport := defaultHTTPTransport.Clone()
// 	proxyTransport.ForceAttemptHTTP2 = false
// 	proxyTransport.TLSClientConfig = &tls.Config{
// 		RootCAs:    certPool,
// 		MinVersion: tls.VersionTLS12,
// 	}

// 	httpErrorLog := log.New(logging.NewFilteredHTTPLogger(), "", 0)

// 	proxy := httputil.NewSingleHostReverseProxy(kubeAPIAddr)
// 	proxy.Transport = proxyTransport
// 	proxy.ErrorLog = httpErrorLog

// 	metricsServer := &http.Server{
// 		ReadHeaderTimeout: 30 * time.Second,
// 		ReadTimeout:       60 * time.Second,
// 		Addr:              options.Addr.Metrics,
// 		Handler:           metrics.NewHandler(promRegistry),
// 		ErrorLog:          httpErrorLog,
// 	}

// 	group.Go(func() error {
// 		err := metricsServer.ListenAndServe()
// 		if errors.Is(err, http.ErrServerClosed) {
// 			return nil
// 		}
// 		return err
// 	})

// 	healthOnlyRouter := gin.New()
// 	healthOnlyRouter.GET("/healthz", func(c *gin.Context) {
// 		c.Status(http.StatusOK)
// 	})

// 	plaintextServer := &http.Server{
// 		ReadHeaderTimeout: 30 * time.Second,
// 		ReadTimeout:       60 * time.Second,
// 		Addr:              options.Addr.HTTP,
// 		Handler:           healthOnlyRouter,
// 		ErrorLog:          httpErrorLog,
// 	}

// 	group.Go(func() error {
// 		err := plaintextServer.ListenAndServe()
// 		if errors.Is(err, http.ErrServerClosed) {
// 			return nil
// 		}
// 		return err
// 	})

// 	authn := newAuthenticator(u.String(), options)
// 	router.Use(
// 		metrics.Middleware(promRegistry),
// 		proxyMiddleware(proxy, authn, k8s.Config.BearerToken),
// 	)
// 	tlsServer := &http.Server{
// 		ReadHeaderTimeout: 30 * time.Second,
// 		ReadTimeout:       60 * time.Second,
// 		Addr:              options.Addr.HTTPS,
// 		TLSConfig:         tlsConfig,
// 		Handler:           router,
// 		ErrorLog:          httpErrorLog,
// 	}

// 	logging.Infof("starting infra connector (%s) - http:%s https:%s metrics:%s", internal.FullVersion(), plaintextServer.Addr, tlsServer.Addr, metricsServer.Addr)

// 	group.Go(func() error {
// 		err = tlsServer.ListenAndServeTLS("", "")
// 		if errors.Is(err, http.ErrServerClosed) {
// 			return nil
// 		}
// 		return err
// 	})

// 	// wait for shutdown signal
// 	<-ctx.Done()

// 	shutdownCtx, shutdownCancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
// 	defer shutdownCancel()
// 	if err := tlsServer.Shutdown(shutdownCtx); err != nil {
// 		logging.L.Warn().Err(err).Msgf("shutdown proxy server")
// 	}
// 	if err := plaintextServer.Close(); err != nil {
// 		logging.L.Warn().Err(err).Msgf("shutdown plaintext server")
// 	}
// 	if err := metricsServer.Close(); err != nil {
// 		logging.L.Warn().Err(err).Msgf("shutdown metrics server")
// 	}

// 	// wait for goroutines to shutdown
// 	err = group.Wait()
// 	if errors.Is(err, context.Canceled) {
// 		return nil
// 	}
// 	return err
// }
