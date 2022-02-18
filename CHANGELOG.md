# Changelog

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
