# Changelog

## [0.8.0](https://github.com/infrahq/infra/compare/v0.7.0...v0.8.0) (2022-03-30)


### ⚠ BREAKING CHANGES

* use consistent api time and duration types (#1344)

### Features

* add /debug/pprof handlers for debugging ([10e7ab0](https://github.com/infrahq/infra/commit/10e7ab05e92553cf7c9931a953de72a75c658b5a))
* default grants with no role to "connect" permission ([#1309](https://github.com/infrahq/infra/issues/1309)) ([3c4a8a9](https://github.com/infrahq/infra/commit/3c4a8a92c5649745b839c5f0e8b1d7da9fa8cb66))
* use consistent api time and duration types ([#1344](https://github.com/infrahq/infra/issues/1344)) ([4cd9c81](https://github.com/infrahq/infra/commit/4cd9c8119d9958b1da5443b4f5fd531c6203d3f3))


### Bug Fixes

* access key extension and deadline no longer optional ([#1386](https://github.com/infrahq/infra/issues/1386)) ([7d593c4](https://github.com/infrahq/infra/commit/7d593c4862a260980f15eef6c39d54065da53d0e))
* cert generation ([9a543a8](https://github.com/infrahq/infra/commit/9a543a8bb4fd544c69bbdb862f21226504193db8))
* **cli:** add test and fix bugs in logout ([e5c280f](https://github.com/infrahq/infra/commit/e5c280f44bcf49776209fc0b7cd1b69505b803ad))
* **cmd:** check login access key is non-empty ([d1eacfc](https://github.com/infrahq/infra/commit/d1eacfc5b515e19b29efe183e2ac2e9b5b20d6ff))
* **cmd:** email validation when adding ids ([506f2c6](https://github.com/infrahq/infra/commit/506f2c65faef9cbc2c581f8106a8f6e70919b68b))
* do not force http on proxy transport ([1db5f25](https://github.com/infrahq/infra/commit/1db5f2579c39f4426175a7879390bd1a02f40a3c))
* do not recreate connector cert if exists ([a1dc9cb](https://github.com/infrahq/infra/commit/a1dc9cbe208d72fca11c170350601fcb18e56429))
* dont use custom tls verification logic in connector ([#1347](https://github.com/infrahq/infra/issues/1347)) ([e96b31c](https://github.com/infrahq/infra/commit/e96b31cd74d396fd5d1851ac5dbb5cd6f6ee2800))
* generation of openapi spec when there are no changes ([#1304](https://github.com/infrahq/infra/issues/1304)) ([67d0240](https://github.com/infrahq/infra/commit/67d02408bb155d450e883b179d22f932bafae96c))
* http.Transport not using reasonable defaults ([4405001](https://github.com/infrahq/infra/commit/4405001336ad032059e24ac2d1e623f6884596e2))
* invalid name test failing randomly ([eb300ec](https://github.com/infrahq/infra/commit/eb300ec4e35281126e4da09225b552299e989ba1))
* k8s connector should ignore "connect" grants ([#1363](https://github.com/infrahq/infra/issues/1363)) ([c170a09](https://github.com/infrahq/infra/commit/c170a095cb620e981a0ecaa63b26f3d1aa26ea13))
* k8s connector: remove provider name prefix ([#1370](https://github.com/infrahq/infra/issues/1370)) ([28e404a](https://github.com/infrahq/infra/commit/28e404a3fd1cb3bfb3f6d4d38e63da238b92fc00))
* note that connector takes time to initialize ([#1343](https://github.com/infrahq/infra/issues/1343)) ([#1358](https://github.com/infrahq/infra/issues/1358)) ([a89b79b](https://github.com/infrahq/infra/commit/a89b79b49275a90afe1ad42e574ce5ba49e9aeb6))
* only init schema if it's never been done ([#1397](https://github.com/infrahq/infra/issues/1397)) ([0c863ee](https://github.com/infrahq/infra/commit/0c863eef8434efeb41b9e9f245be8281f483182e))
* recreate access key if parts differ ([c6b50b7](https://github.com/infrahq/infra/commit/c6b50b76a5ab0c91773501f1c6da76984474e0b0))
* rename force flag to purge ([9c5112c](https://github.com/infrahq/infra/commit/9c5112c5319fea0caf3584709e1efb03b5ac39df))
* small godoc and logging fixes ([21e0b6e](https://github.com/infrahq/infra/commit/21e0b6ed788b41abb9cf65c2780c85ef4f5674f4))

## [0.7.0](https://github.com/infrahq/infra/compare/v0.6.1...v0.7.0) (2022-03-24)


### ⚠ BREAKING CHANGES

* **cli:** some CLI short flags were removed.

### Features

* **connector:** add support for bringing in custom certificates ([2650769](https://github.com/infrahq/infra/commit/26507694d625ab81ae0cbfbbc88630208be64ef8))


### Bug Fixes

* **cmd:** destinations add takes dot notation as input ([6eb393c](https://github.com/infrahq/infra/commit/6eb393c1ce848ed44b3da26559fec7b9804a00b7))
* do not fail on no groups or ref from oidc ([#1281](https://github.com/infrahq/infra/issues/1281)) ([bbc0247](https://github.com/infrahq/infra/commit/bbc02474136a7763840dc2151f816380b3bf825d))
* give users infra grant on creation ([#1295](https://github.com/infrahq/infra/issues/1295)) ([c472b41](https://github.com/infrahq/infra/commit/c472b410eb00b14023abf16542ef80ca970963e7))
* update config for styled component ([#1307](https://github.com/infrahq/infra/issues/1307)) ([91aa01f](https://github.com/infrahq/infra/commit/91aa01ff61761caf6508c60717194ba6d659eac2))


### Styles

* **cli:** remove some short flags from CLI commands ([4f1ef00](https://github.com/infrahq/infra/commit/4f1ef00403940313e852bfcf7c83e0ea7ae31f71))

### [0.6.1](https://github.com/infrahq/infra/compare/v0.6.0...v0.6.1) (2022-03-21)


### Features

* Local Identity CLI ([#1269](https://github.com/infrahq/infra/issues/1269)) ([a5159b2](https://github.com/infrahq/infra/commit/a5159b27043d2f55955c604b619a01015a0677f1))


### Bug Fixes

* make openapi generation deterministic ([#1270](https://github.com/infrahq/infra/issues/1270)) ([d371b56](https://github.com/infrahq/infra/commit/d371b56c8723544e6117dc8b98d8b82516a8bc0c))

## [0.6.0](https://github.com/infrahq/infra/compare/v0.5.12...v0.6.0) (2022-03-19)


### ⚠ BREAKING CHANGES

* **fix:** rename engine to connector

### Features

* local infra users ([#1223](https://github.com/infrahq/infra/issues/1223)) ([67b1f90](https://github.com/infrahq/infra/commit/67b1f90e5c63ad1134e8d69162b82960e651d070))
* rename engine to connector ([#1229](https://github.com/infrahq/infra/issues/1229)) ([c0ebc01](https://github.com/infrahq/infra/commit/c0ebc018cd1b331ce629b44875734644aec0e22a))


### Bug Fixes

* **ci:** bump helm chart version with release ([#1239](https://github.com/infrahq/infra/issues/1239)) ([ac09090](https://github.com/infrahq/infra/commit/ac090907905c35b3044adcafd6cefc9f5ee29923))
* client should skip json parse body on unknown errors ([#1242](https://github.com/infrahq/infra/issues/1242)) ([8b1e805](https://github.com/infrahq/infra/commit/8b1e805ce280a4f2692e03f8c893d1b185f94063))
* dont store duplicate grants ([#1228](https://github.com/infrahq/infra/issues/1228)) ([aa49ec6](https://github.com/infrahq/infra/commit/aa49ec6f24381532884782020eab9b72d36340b4))
* hide infra provider from login CLI ([#1240](https://github.com/infrahq/infra/issues/1240)) ([1725ac5](https://github.com/infrahq/infra/commit/1725ac5a9dc3da11ccce7a168aab7d620aa5ad4d))
* remove unused package-lock.json ([7e5b344](https://github.com/infrahq/infra/commit/7e5b34446afc6641291fae706c579d4c67c3b350))


### Miscellaneous Chores

* **fix:** rename engine to connector ([#1244](https://github.com/infrahq/infra/issues/1244)) ([23d89a6](https://github.com/infrahq/infra/commit/23d89a6ee93ffbafd4b508e78aa336c11a29b1a5))

### [0.5.12](https://github.com/infrahq/infra/compare/v0.5.11...v0.5.12) (2022-03-17)


### Features

* adding okta provider ([#1164](https://github.com/infrahq/infra/issues/1164)) ([4880990](https://github.com/infrahq/infra/commit/4880990fc8814899a6afb06d01371fe632da8789))
* **helm:** disable engine install by default ([416239d](https://github.com/infrahq/infra/commit/416239dbddf6ff947f87d5bfac26350e8d88c22b))
* **helm:** infer server.enabled ([166eb4e](https://github.com/infrahq/infra/commit/166eb4e17046005920bc1145cceb01d8d924968f))
* infer server if configured for localhost ([93c6fca](https://github.com/infrahq/infra/commit/93c6fca03b9430f7365998b9effbb11b7f30f34e))


### Bug Fixes

* **cli:** outputs unique list of grants for each dest ([#1140](https://github.com/infrahq/infra/issues/1140)) ([158bc9c](https://github.com/infrahq/infra/commit/158bc9ca7d836d0c324349e5c9ec8d5af2d630b3))
* **cmd:** use single field to cache client identity ([567e664](https://github.com/infrahq/infra/commit/567e664051004bda5ecc24051b619e133bbce16d))
* consistent version format ([31ecc2a](https://github.com/infrahq/infra/commit/31ecc2a025f8ddb9fea54bf2efe091682f5303ab))
* **data:** discard gorm logs ([fe7ec71](https://github.com/infrahq/infra/commit/fe7ec7149fabe49aa76a628c9c7da9d492271f1c))
* **engine:** remove Recovery middleware ([d24e8e2](https://github.com/infrahq/infra/commit/d24e8e2a4b17393e92ab3b292341a70e397303ed))
* **helm:** remove engine ingress values ([48f448e](https://github.com/infrahq/infra/commit/48f448ecda803a825158349769d3de962a4f4196))
* improves UX around validation errors ([#1209](https://github.com/infrahq/infra/issues/1209)) ([becacb0](https://github.com/infrahq/infra/commit/becacb06eb63b2cafe5425b4ca7105c9e8c403ba))
* nil error log when destination refreshed ([#1185](https://github.com/infrahq/infra/issues/1185)) ([ec73d52](https://github.com/infrahq/infra/commit/ec73d5222519cb477f13ee0a62472add08686aed))
* **server:** fix import access key when key part changes ([7977e08](https://github.com/infrahq/infra/commit/7977e08c0ded403abc3b204d5a5df06d93068cca))
* update grant examples ([#1222](https://github.com/infrahq/infra/issues/1222)) ([45617cc](https://github.com/infrahq/infra/commit/45617cc36af420fff1f6b97bf7e9c26138bf310b))

### [0.5.11](https://github.com/infrahq/infra/compare/v0.5.10...v0.5.11) (2022-03-12)


### Bug Fixes

* gitignore ignores helm/charts/infra ([c413389](https://github.com/infrahq/infra/commit/c413389869148133913ce4c9384274765329bd35))
* update infra kube config tokens command ([#1170](https://github.com/infrahq/infra/issues/1170)) ([bb8e654](https://github.com/infrahq/infra/commit/bb8e654316349eef0a15ea7ee4d0e2a73b6368e8))
* use correct readme uploader action ([176567d](https://github.com/infrahq/infra/commit/176567dfb6111bf974402ce3093b8f822adb34c2))

### [0.5.10](https://github.com/infrahq/infra/compare/v0.5.9...v0.5.10) (2022-03-04)


### Bug Fixes

* **cli:** logged out error for most commands ([#1130](https://github.com/infrahq/infra/issues/1130)) ([93392aa](https://github.com/infrahq/infra/commit/93392aadb6169265f0bfa7c6216189c7b12750ac))
* db locked message in sqlite ([#1147](https://github.com/infrahq/infra/issues/1147)) ([838e842](https://github.com/infrahq/infra/commit/838e842a2b2d59c88bf3ee6bd86801defd085244))

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
