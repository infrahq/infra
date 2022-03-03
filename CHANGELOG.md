# Changelog

### [0.5.9](https://github.com/infrahq/infra/compare/v0.5.8...v0.5.9) (2022-03-03)


### Features

* add docker-compose.yml ([8399b7c](https://github.com/infrahq/infra/commit/8399b7ccf4f249c520f03acac10faac4069eccf5))
* add one-time setup endpoint ([8bcfe2f](https://github.com/infrahq/infra/commit/8bcfe2f2fcf132b14d6cc38725d851d071beaed5))
* Certificates management ([#1086](https://github.com/infrahq/infra/issues/1086)) ([09802a3](https://github.com/infrahq/infra/commit/09802a3b7563f95ef2d52e0ff9b913b3a2d0a259))
* **cmd:** setup if required during CLI login ([b7b3b84](https://github.com/infrahq/infra/commit/b7b3b841eb2c5463748d98e748a2586c526de998))
* **helm:** single helm chart ([a1ffddd](https://github.com/infrahq/infra/commit/a1ffddd5950430001b2772facd68e50082e067d6))
* remove infra permissions ([#855](https://github.com/infrahq/infra/issues/855)) ([#1085](https://github.com/infrahq/infra/issues/1085)) ([fb7f3fe](https://github.com/infrahq/infra/commit/fb7f3fe3a82e0e019e732a2e97876ada878cd11d))
* switching back to js ([#1111](https://github.com/infrahq/infra/issues/1111)) ([578d874](https://github.com/infrahq/infra/commit/578d8749acf87b2fdc1060b404bfd98d48aeaa8e))


### Bug Fixes

* add start up log to server/engine ([ba200e0](https://github.com/infrahq/infra/commit/ba200e0d705c2ef2d0082229d4e23163070c41c4))
* better client error messages ([#1126](https://github.com/infrahq/infra/issues/1126)) ([7c37bcb](https://github.com/infrahq/infra/commit/7c37bcb08234206abdd4d0c4ef478d5f57ee9a3c))
* clean up tls cert error display in cli ([#1032](https://github.com/infrahq/infra/issues/1032)) ([4d14842](https://github.com/infrahq/infra/commit/4d14842ff0407986da7d22dbe58b537dfae85b37))
* cli required parameter validation ([#1045](https://github.com/infrahq/infra/issues/1045)) ([c2938c5](https://github.com/infrahq/infra/commit/c2938c5f96cbf8d47e84029b8e73009475ea71ac))
* **cli:** info checks if user is logged in ([#1100](https://github.com/infrahq/infra/issues/1100)) ([bbae6d9](https://github.com/infrahq/infra/commit/bbae6d9022db86e6ad8fd23efb140522c1130b86))
* **cli:** small if-statement flip for machine validation options ([#1096](https://github.com/infrahq/infra/issues/1096)) ([e4c7b4a](https://github.com/infrahq/infra/commit/e4c7b4a58fd82ad3584ca67433ccb2f409652d7a))
* engine connection updating ([#955](https://github.com/infrahq/infra/issues/955)) ([#1131](https://github.com/infrahq/infra/issues/1131)) ([b1cea2b](https://github.com/infrahq/infra/commit/b1cea2ba27c85dbe434f8bc05ade8b864bfd080b))
* http request logging ([cbcb13b](https://github.com/infrahq/infra/commit/cbcb13b54dd0f83253a739839acf1d88591bb761))
* include provider in jwt claim ([#1132](https://github.com/infrahq/infra/issues/1132)) ([43e979c](https://github.com/infrahq/infra/commit/43e979c4b454a40d4dbda130b9a3a446d290b9b7))
* infra casing ([#1043](https://github.com/infrahq/infra/issues/1043)) ([8a1da13](https://github.com/infrahq/infra/commit/8a1da1378676d5a543b414f8382968fdf1826210))
* load config directly into db ([ea515cc](https://github.com/infrahq/infra/commit/ea515cc8fe79209773bd8b5dc60e62ecc6945ae6))
* remove cmd dependency on access ([#1117](https://github.com/infrahq/infra/issues/1117)) ([c745f54](https://github.com/infrahq/infra/commit/c745f544f90fb658ab8b938a433418abb1b28cb3))
* remove user token by issued for on logout ([#1066](https://github.com/infrahq/infra/issues/1066)) ([882b5ce](https://github.com/infrahq/infra/commit/882b5ceb0c57eadca1278974c386ed248e87b1dc))
* replace logging.Logger with logging.WrappedLogger ([c3c047b](https://github.com/infrahq/infra/commit/c3c047bd5e5014e759460328593cb62939d8d5b7))
* resolves issue with panic around unexpected grant syntax ([#1113](https://github.com/infrahq/infra/issues/1113)) ([269454b](https://github.com/infrahq/infra/commit/269454b10b461dd181e4cad806268f031426f82a))
* update .gitignore to include infra binary ([#1072](https://github.com/infrahq/infra/issues/1072)) ([cfa2248](https://github.com/infrahq/infra/commit/cfa2248d4d97f772d3182e95d92aa5b81d3f6a17))
* when logout it should redirect back to login page ([#1089](https://github.com/infrahq/infra/issues/1089)) ([c15dab1](https://github.com/infrahq/infra/commit/c15dab114bf459956054f2254158fd95d4a254be))

### [0.5.8](https://github.com/infrahq/infra/compare/v0.5.7...v0.5.8) (2022-02-18)


### Bug Fixes

* **ci:** deep clone ([bce3f2e](https://github.com/infrahq/infra/commit/bce3f2eee128aed913c7cd427b2a60648be19b6b))

### [0.5.7](https://github.com/infrahq/infra/compare/v0.5.6...v0.5.7) (2022-02-18)


### Features

* **engine:** add metrics middleware ([1f8a349](https://github.com/infrahq/infra/commit/1f8a349db3900a9edfe2fec5041891c951151134))


### Bug Fixes

* Change default logging on server errors to Error instead of debug ([#1011](https://github.com/infrahq/infra/issues/1011)) ([df2f5e6](https://github.com/infrahq/infra/commit/df2f5e6b267cd8fd8d4d3110c8da78d825d747aa))
* **ci:** fix release_created/name output path ([fae4810](https://github.com/infrahq/infra/commit/fae4810c74ab516d33ff445a84ce78fbedcdeb29))
* do not overwrite default name when generating auto engine name ([#942](https://github.com/infrahq/infra/issues/942)) ([#1029](https://github.com/infrahq/infra/issues/1029)) ([09a47d2](https://github.com/infrahq/infra/commit/09a47d2568a1ea12ec7899f026ba07015e6a642d))
* **engine:** dont skip tls verify by default ([#917](https://github.com/infrahq/infra/issues/917)) ([8f69a16](https://github.com/infrahq/infra/commit/8f69a1605f2e9002241bcbcea8f315f9830d1a7d))
* generate access key name if needed ([#1006](https://github.com/infrahq/infra/issues/1006)) ([1346b50](https://github.com/infrahq/infra/commit/1346b50af1bfae7f4812d31b124cfd6e02163ba9))
* **helm:** engine templates should reference engine ([12f0963](https://github.com/infrahq/infra/commit/12f09630ed76986becd5fb909aad6748850f69a6))
* **helm:** podAnnotation breaking helm templating ([c4255af](https://github.com/infrahq/infra/commit/c4255affdc50ef9a13e9e41e6f44b4892b0ed096))
* set destination ID on update ([fd30d53](https://github.com/infrahq/infra/commit/fd30d539a0646ac7756e3d9f2fc728413660367b))
* update login flow for access key and multiple redirect URLs ([#1014](https://github.com/infrahq/infra/issues/1014)) ([b05b5ff](https://github.com/infrahq/infra/commit/b05b5ff7dbc76f7a4e88ceea9a3dc1d77304f299))

### [0.5.6](https://github.com/infrahq/infra/compare/v0.5.5...v0.5.6) (2022-02-17)


### Features

* **ci:** add commitlint action and configs ([dd180f1](https://github.com/infrahq/infra/commit/dd180f1b8640724e246897584e79fd436927a46f))
* **cmd:** add soft and hard logout ([91edc38](https://github.com/infrahq/infra/commit/91edc38d5ac006fbe0d74afd8071c52d7b59536e))
* **helm:** allow users to side load access keys ([d16a858](https://github.com/infrahq/infra/commit/d16a8584b48bb965111e95efb20bce4b2a0ba195))
* machine authn w/ access keys ([#971](https://github.com/infrahq/infra/issues/971)) ([352a6ee](https://github.com/infrahq/infra/commit/352a6ee8939f1f9553feb58a2348fa360b685f80))
* use secret storage in engine ([a312272](https://github.com/infrahq/infra/commit/a31227264b53e63cc8129393660d915286b15dc8))


### Bug Fixes

* **ci:** compare boolean values ([e4b69cd](https://github.com/infrahq/infra/commit/e4b69cdb38855355c56cd470bcf10d126432b4d6))
* **ci:** setup docker buildx ([cbc9e73](https://github.com/infrahq/infra/commit/cbc9e73432ed73aee2a9744a2f7a968f5a8c76f7))
* **ci:** use personal access token ([594e17e](https://github.com/infrahq/infra/commit/594e17e783942d0c8639ff35c854bfc20d89999a))
* **helm:** fix accessKey error check in  engine NOTES.txt ([#1002](https://github.com/infrahq/infra/issues/1002)) ([82d13e0](https://github.com/infrahq/infra/commit/82d13e00a71e50a109a6eff8e8b736835e17f367))
* **helm:** move access keys to config ([e487cd3](https://github.com/infrahq/infra/commit/e487cd3982ea7e4adf411fcdcec7f0d6fe0010e2))
* **helm:** server config.import templating ([04af868](https://github.com/infrahq/infra/commit/04af8685edd015761d54cb38f92474c813f00a5e))
* **helm:** server db config hints ([bd2a628](https://github.com/infrahq/infra/commit/bd2a628dfc1ef376671a31209e7697969d0fe248))
