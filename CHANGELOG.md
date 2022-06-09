# Changelog

## [0.13.2](https://github.com/infrahq/infra/compare/v0.13.1...v0.13.2) (2022-06-09)


### Features

* add list users in group api + groups list cli ([#2193](https://github.com/infrahq/infra/issues/2193)) ([9fcfa94](https://github.com/infrahq/infra/commit/9fcfa9422b006dd2cb84b66205f072489d0515ba))
* add more default cluster roles ([7771c50](https://github.com/infrahq/infra/commit/7771c50f20a5146684224e3b1e44288c56f84262))
* detect unique constraint violation ([be062c9](https://github.com/infrahq/infra/commit/be062c9b941b7fccfd1d9b348909cab89e8aa6b2))
* get user api should return names of providers user belongs to ([#2219](https://github.com/infrahq/infra/issues/2219)) ([17adab4](https://github.com/infrahq/infra/commit/17adab46f48b0de73464fbb233f6da19f742a2ab))


### Bug Fixes

* allow infra prefix for `infra use` ([#2206](https://github.com/infrahq/infra/issues/2206)) ([a20b998](https://github.com/infrahq/infra/commit/a20b9981fab07261b4d71e9b06c89921d70d490b))
* auth and signup redirects ([#2183](https://github.com/infrahq/infra/issues/2183)) ([e4ed20c](https://github.com/infrahq/infra/commit/e4ed20c056f8ead94a1cf1bdf3637defdf796bf6))
* create kubeconfig dir ([c0cb835](https://github.com/infrahq/infra/commit/c0cb83529e86da2d633114cf5244ee6134ee29e2))
* deadlock before ingress ready ([#2202](https://github.com/infrahq/infra/issues/2202)) ([b273104](https://github.com/infrahq/infra/commit/b27310414840a5a3cc663bc70fa40a5bff36ab4a))
* destination CA field encoding ([#2217](https://github.com/infrahq/infra/issues/2217)) ([bbc97d5](https://github.com/infrahq/infra/commit/bbc97d5d58da232cb014f70be8de63651b431ad4))
* do not warn about the connector identity ([#2245](https://github.com/infrahq/infra/issues/2245)) ([b80ef64](https://github.com/infrahq/infra/commit/b80ef64c5d0ce2c1aee471a4b20e095a5bb6cace))
* fix auth ui redirects for new otp users ([#2091](https://github.com/infrahq/infra/issues/2091)) ([e8744ad](https://github.com/infrahq/infra/commit/e8744ad82840780e8c1720a456c11fcf88f64df8))
* fix replica count variable name ([#2190](https://github.com/infrahq/infra/issues/2190)) ([d2c06e6](https://github.com/infrahq/infra/commit/d2c06e60469aeb7128f7419a1d5a4023e5277103))
* handle undefined when first load the grants ([#2187](https://github.com/infrahq/infra/issues/2187)) ([c86e1fe](https://github.com/infrahq/infra/commit/c86e1fe6a219a06ff4c7e01a33bd71bccbfc14f3))
* infra default role is no role ([#2148](https://github.com/infrahq/infra/issues/2148)) ([e40a202](https://github.com/infrahq/infra/commit/e40a20298815988da316128c9b89573ec8faffe4))
* loading dbUsername ([b81a4ab](https://github.com/infrahq/infra/commit/b81a4abd1b79a9b4ee80b9b4b30c0ed5e00f5d62))
* make testuserscmd more resilient ([#2215](https://github.com/infrahq/infra/issues/2215)) ([2cf85f8](https://github.com/infrahq/infra/commit/2cf85f82b0e5a294c032ae299ad2c3793cab3ff6))
* modify set-password help text ([#2212](https://github.com/infrahq/infra/issues/2212)) ([ac0b7ec](https://github.com/infrahq/infra/commit/ac0b7ec992c982bbca42aff3f4ea82acff3868d0))
* profile-icon component refactor ([#2150](https://github.com/infrahq/infra/issues/2150)) ([b84dcd7](https://github.com/infrahq/infra/commit/b84dcd7fe00f2b5ff7a227358884085da6fbeda1))
* release build should not contain dev ([f26db74](https://github.com/infrahq/infra/commit/f26db74ea62fb61f5742a00fa8ed26ca22f8eb91))
* remove identity from CLI help ([#2172](https://github.com/infrahq/infra/issues/2172)) ([3350a80](https://github.com/infrahq/infra/commit/3350a807757714564e0c4b2c610d0e99f154bbc7))
* require user to have an email ([#2181](https://github.com/infrahq/infra/issues/2181)) ([3231437](https://github.com/infrahq/infra/commit/3231437ff576a994eec93353f811c988976fd281))
* ui bugs ([#2120](https://github.com/infrahq/infra/issues/2120)) ([128bd01](https://github.com/infrahq/infra/commit/128bd017afd54b104ff8c85646f2e43901a4ba84))
* ui bugs ([#2142](https://github.com/infrahq/infra/issues/2142)) ([66ef1b9](https://github.com/infrahq/infra/commit/66ef1b964ddbf92f1f861ffeabb151fd4b95c9ae))
* ui crash on null created date ([#2216](https://github.com/infrahq/infra/issues/2216)) ([c8c8db5](https://github.com/infrahq/infra/commit/c8c8db538b44f8a5b2dc333f9628a4655b125ba1))
* ui props being passed when using authrequired ([#2169](https://github.com/infrahq/infra/issues/2169)) ([65a36eb](https://github.com/infrahq/infra/commit/65a36ebf527f1a28217d7016551cad7fbc338cb3))
* ui scrolling ([#2184](https://github.com/infrahq/infra/issues/2184)) ([ac4af12](https://github.com/infrahq/infra/commit/ac4af128dc474b32bdd7fd83ffc9403c477a39c1))

### [0.13.1](https://github.com/infrahq/infra/compare/v0.13.0...v0.13.1) (2022-05-27)


### Features

* add force flag to grants add ([67fd367](https://github.com/infrahq/infra/commit/67fd36790fa6456f2448e65cfaa1e6d9e0c3a373))
* agent to keep kube config up to date ([#2013](https://github.com/infrahq/infra/issues/2013)) ([defcf4e](https://github.com/infrahq/infra/commit/defcf4e448b885a8f01c0fef5313b7136057a41a))
* enable server UI by default ([c5ec0fb](https://github.com/infrahq/infra/commit/c5ec0fb787a47f40e6fb5144bd325a29597ed4b5))
* error on removal if resource does not exist ([0af2f52](https://github.com/infrahq/infra/commit/0af2f521e45583a58fd5b9b4bfb762ccb74077af))
* update docs ([ae47563](https://github.com/infrahq/infra/commit/ae475639aadff52bbc309606df1199bd2ab0a8ba))


### Bug Fixes

* a few bugs on grants and deletion ([#2061](https://github.com/infrahq/infra/issues/2061)) ([0537f93](https://github.com/infrahq/infra/commit/0537f939ebddf64792ce8eb55e4afce1d5b0f4eb))
* add singular alias for users ([#1951](https://github.com/infrahq/infra/issues/1951)) ([afc482a](https://github.com/infrahq/infra/commit/afc482a95a7e7549995016ea2cf17eeb71b6afbf))
* allow non email usernames ([#2069](https://github.com/infrahq/infra/issues/2069)) ([f5508ee](https://github.com/infrahq/infra/commit/f5508ee9b160f4e99e6782d375aa120e9d7d40bc))
* allow password and accesskey on config user ([#2037](https://github.com/infrahq/infra/issues/2037)) ([5c04561](https://github.com/infrahq/infra/commit/5c0456163e60256b105bde60f0d1333a0fb3be05))
* client requests should time out ([#1966](https://github.com/infrahq/infra/issues/1966)) ([98aa32f](https://github.com/infrahq/infra/commit/98aa32ffb0966cc4438b5b7392bf8e72f9a4ca1c))
* dashboard loading and redirect states ([#2049](https://github.com/infrahq/infra/issues/2049)) ([815d317](https://github.com/infrahq/infra/commit/815d3173ab197d787f734203bc1e339d8dd192ea))
* do not show --skipTLSVerify in helm command if using https ([#2070](https://github.com/infrahq/infra/issues/2070)) ([c2ac1e6](https://github.com/infrahq/infra/commit/c2ac1e60dbd578861d3359d073f9f805c27de91e))
* docker image build sets correct version string ([2fbef36](https://github.com/infrahq/infra/commit/2fbef3628d5189a0637e68f350a0ab6174b3da8e))
* documentation fixes ([#1960](https://github.com/infrahq/infra/issues/1960)) ([10cb28f](https://github.com/infrahq/infra/commit/10cb28f2043d3da29a8bead8c0a1343e6b6f1949))
* fail fast on duplicate config access key ([#2046](https://github.com/infrahq/infra/issues/2046)) ([42d938f](https://github.com/infrahq/infra/commit/42d938f64ea943cc14cbb091b13bfd1a0702f003))
* fix make helm target ([#2092](https://github.com/infrahq/infra/issues/2092)) ([c76e1b3](https://github.com/infrahq/infra/commit/c76e1b3daad15772b5350a000eb73777be4f0394))
* fix prerelease docker build ([#2035](https://github.com/infrahq/infra/issues/2035)) ([db3adda](https://github.com/infrahq/infra/commit/db3addab66d3ae94f1621b18a7a32b229bd6ba86))
* fix ui build in release action ([#2044](https://github.com/infrahq/infra/issues/2044)) ([febe197](https://github.com/infrahq/infra/commit/febe19702121acc823daeceeebda926124a1743c))
* keep jwt nonce for version compatibility ([#2086](https://github.com/infrahq/infra/issues/2086)) ([172641d](https://github.com/infrahq/infra/commit/172641d0ae5483e0071454b1794e06e6b143ed4e))
* missing delete modal setOpen hooks ([#2052](https://github.com/infrahq/infra/issues/2052)) ([09d06da](https://github.com/infrahq/infra/commit/09d06dae62e498bc856185fa9bd688b7babe146c))
* no http path, method in errors ([2e8d94c](https://github.com/infrahq/infra/commit/2e8d94c989d488e64b6663e44cc20d0eaa69c2e3))
* openapi gen should not set -dev version in release branches ([0463e28](https://github.com/infrahq/infra/commit/0463e2836226a14953bb98ff7930e3411b8d61bf))
* optional grant role ([b8f281d](https://github.com/infrahq/infra/commit/b8f281d5dabe0bdb4dabeb925a2af2aa415c45a8))
* pass the userID arg to the request ([#2097](https://github.com/infrahq/infra/issues/2097)) ([de675ce](https://github.com/infrahq/infra/commit/de675ce856b4e2072bb30b5554834d45b160d532))
* prevent invalid cross-device link error ([f24011d](https://github.com/infrahq/infra/commit/f24011dd5d8e934e02c385498354378f5c01699f))
* properly account for .items in destination connectivity check ([#2059](https://github.com/infrahq/infra/issues/2059)) ([b7a466e](https://github.com/infrahq/infra/commit/b7a466eaf922b3b0cb7c0a23626a8c455430fa31))
* readme-compatible openapi spec ([#1954](https://github.com/infrahq/infra/issues/1954)) ([034b110](https://github.com/infrahq/infra/commit/034b11082edbd59067c954f03d731a85291c881f))
* remove BUILDVERSION from makefile ([aba1048](https://github.com/infrahq/infra/commit/aba10484da46358a347c61bef3316ea6099a4115))
* remove deprecated routes from openapi doc ([20f38d4](https://github.com/infrahq/infra/commit/20f38d4c76a32c2ce73e52984721839c3bff9891))
* remove incorrect connection field until we update the api to support this ([#2062](https://github.com/infrahq/infra/issues/2062)) ([a38cdf8](https://github.com/infrahq/infra/commit/a38cdf82a323eeae48fc11d919e3524b098e1a39))
* remove infra user role from docs ([#1961](https://github.com/infrahq/infra/issues/1961)) ([3033a3b](https://github.com/infrahq/infra/commit/3033a3b61730b93a4739a18f8093153f02bef08d))
* remove redirect to /login when logging in with otp ([#2071](https://github.com/infrahq/infra/issues/2071)) ([cf5140e](https://github.com/infrahq/infra/commit/cf5140e0582b68cfcc9fbdb6e941cfb84894cc03))
* remove unneeded jwt nonce ([#2015](https://github.com/infrahq/infra/issues/2015)) ([157e558](https://github.com/infrahq/infra/commit/157e55802423ed25cc5a0f2c907c2b30125c18a4))
* rename /v1/ to /api/ in docs concepts ([#2093](https://github.com/infrahq/infra/issues/2093)) ([5bca519](https://github.com/infrahq/infra/commit/5bca519410e37e9b5ce3c1d6e2c0328e78afa1f8))
* revert "improve: support cli error struct  ([#1936](https://github.com/infrahq/infra/issues/1936))" ([27d724e](https://github.com/infrahq/infra/commit/27d724eb79a6627e1edfc614e1a3c8b5db5725f4))
* run standard and include it in ui tests ([#2055](https://github.com/infrahq/infra/issues/2055)) ([5b1e90e](https://github.com/infrahq/infra/commit/5b1e90e41b502398d6340d1529d46e30d123039b))
* set default key name from identity and key id ([#1985](https://github.com/infrahq/infra/issues/1985)) ([402b6c7](https://github.com/infrahq/infra/commit/402b6c7f68bd00eaf0018c9f2e6ecdbd8b03477b))
* ui api item caching and login flow ([#2057](https://github.com/infrahq/infra/issues/2057)) ([ce069bc](https://github.com/infrahq/infra/commit/ce069bc58f854c17ef0e66ed4d143ebf4f56b67e))
* ui infrastructure page title ([#2094](https://github.com/infrahq/infra/issues/2094)) ([a0ad5cb](https://github.com/infrahq/infra/commit/a0ad5cbd63c45b55c7fdafda0082e945b1e79bc9))
* ui showing NaN in relative time fields ([#2058](https://github.com/infrahq/infra/issues/2058)) ([55e0143](https://github.com/infrahq/infra/commit/55e01436428eb42ba741333e4c7d3d30f7799318))
* ui users add fullscreen layout fixes ([#2050](https://github.com/infrahq/infra/issues/2050)) ([da2e576](https://github.com/infrahq/infra/commit/da2e5761367124b6c12dd0bb8a6f3ac356434cc8))
* unset CurrentContext on infra logout ([963e5d6](https://github.com/infrahq/infra/commit/963e5d6274d94899e4654af849d0403d889ea6a1))
* use GITHUB_HEAD_REF instead of GITHUB_REF_NAME ([c39cbc0](https://github.com/infrahq/infra/commit/c39cbc08d4278760387de5bf2e425320d2d653a0))
* users icon frame size ([#2048](https://github.com/infrahq/infra/issues/2048)) ([4600ab4](https://github.com/infrahq/infra/commit/4600ab436efe5314302bc7836deff2d322343d3e))

## [0.13.0](https://github.com/infrahq/infra/compare/v0.12.2...v0.13.0) (2022-05-12)


### ⚠ BREAKING CHANGES

* remove `kubernetes.` prefix from destinations (#1849)

### Features

* add api versioning with request/response migrations ([#1884](https://github.com/infrahq/infra/issues/1884)) ([41527b8](https://github.com/infrahq/infra/commit/41527b8e82ee95883d2bfdd05629491cccbda812))
* add api versioning with request/response migrations ([#1884](https://github.com/infrahq/infra/issues/1884)) ([#1911](https://github.com/infrahq/infra/issues/1911)) ([a2c0c12](https://github.com/infrahq/infra/commit/a2c0c12e2131350919f46c84d91ea6fa58e0e3d4))
* add resources/roles to destinations api ([71598e0](https://github.com/infrahq/infra/commit/71598e0cc94e90abf1fadc562186f8ac2ecce4d0))
* **cmd:** check namespace/role exists in grant add ([e257ba2](https://github.com/infrahq/infra/commit/e257ba205a7b00f50a1e2f03b3980b9ff995bd0e))
* dereference polymorhpic IDs in grant responses ([2b82143](https://github.com/infrahq/infra/commit/2b82143f0ce712f4f308f6e39723ea59b46687ef))
* introduce cli errors ([#1786](https://github.com/infrahq/infra/issues/1786)) ([d7dab70](https://github.com/infrahq/infra/commit/d7dab707cfaec8e547e5a8249869bc68aee1e491))
* list responses as objects ([406915f](https://github.com/infrahq/infra/commit/406915fb1e8e45e4b354f589d21aae2e84cd31f6))
* move info and version from flags to other commands ([#1864](https://github.com/infrahq/infra/issues/1864)) ([cf9e073](https://github.com/infrahq/infra/commit/cf9e073ee99fbabdf05a263576729f09684f1ae4))
* remove polymorphic id from login response ([3aa34e5](https://github.com/infrahq/infra/commit/3aa34e5fdf2a30cbe697315366a7fa5c2a0dabd5))
* update clients to use list responses ([b679426](https://github.com/infrahq/infra/commit/b6794260e8f270b322378f48acef43b59f99537f))
* use email name for authinfo inside of kube ([#1852](https://github.com/infrahq/infra/issues/1852)) ([e93cd84](https://github.com/infrahq/infra/commit/e93cd84c06499780b77cade085f8fa3b5dffa4b5))


### Bug Fixes

* `created_at` no longer zeros on save ([#1893](https://github.com/infrahq/infra/issues/1893)) ([22be9b8](https://github.com/infrahq/infra/commit/22be9b86267879cceb8274fc1017d7a170cb72c3))
* add additional signup check ([703966e](https://github.com/infrahq/infra/commit/703966eca8be62874a77b3e69f373dc5539ef29d))
* add button to add users ([#1826](https://github.com/infrahq/infra/issues/1826)) ([681d79c](https://github.com/infrahq/infra/commit/681d79c7c3f7680874a8d0dbf8879db8d13cd439))
* cleanup telemetry ([#1874](https://github.com/infrahq/infra/issues/1874)) ([69c61ad](https://github.com/infrahq/infra/commit/69c61ada4e9622b68b334c306d38c43fd669fe58))
* **connector:** index out of bounds ([1190126](https://github.com/infrahq/infra/commit/119012655ee95f45fcd82c2c934c198ce3c41dbb))
* disable sign-up before username and pass set ([#1870](https://github.com/infrahq/infra/issues/1870)) ([19d04cc](https://github.com/infrahq/infra/commit/19d04ccc555abd1bf5904bb70e425a77948fdb6c))
* general ui bugs ([#1785](https://github.com/infrahq/infra/issues/1785)) ([5638a0e](https://github.com/infrahq/infra/commit/5638a0ed05aededf84c7ae404f0c4814dceabf99))
* golint 1.6.0 errors ([#1909](https://github.com/infrahq/infra/issues/1909)) ([cf22f66](https://github.com/infrahq/infra/commit/cf22f667b3b2a786b007a1fa972397c6ae5b5fe9))
* issues in documentation to reflect latest changes ([#1950](https://github.com/infrahq/infra/issues/1950)) ([aca1f68](https://github.com/infrahq/infra/commit/aca1f6870d3094004384565106cc47ff8f17179b))
* load identities using name or email ([d72128f](https://github.com/infrahq/infra/commit/d72128f649bffcdd1716ee7156bd335ffacdd303))
* logout failure and error on safari ([#1799](https://github.com/infrahq/infra/issues/1799)) ([2a711dc](https://github.com/infrahq/infra/commit/2a711dc710fba8341df9238b09314031c154ffac))
* logout should always succeed ([#1926](https://github.com/infrahq/infra/issues/1926)) ([73b2a84](https://github.com/infrahq/infra/commit/73b2a8444919b0532a4384fce9d171a93de66e27))
* make dev using wrong version ([a9b1dd7](https://github.com/infrahq/infra/commit/a9b1dd71bd6acbf454ea5b701d53804c00cf8c55))
* make setting secrets in env clear ([#1869](https://github.com/infrahq/infra/issues/1869)) ([6fd0552](https://github.com/infrahq/infra/commit/6fd0552aaffa96af8178862c8100e7d8bbbb53bf))
* migrate grants subject ([#1935](https://github.com/infrahq/infra/issues/1935)) ([785dae4](https://github.com/infrahq/infra/commit/785dae4717634f251f296d016d25f7abf8efe9b4))
* minor presentational ui bugs ([#1876](https://github.com/infrahq/infra/issues/1876)) ([8d9a4ed](https://github.com/infrahq/infra/commit/8d9a4ed3d01e39b4c0a1409172ccc6172446faf6))
* multi providers ui bug ([#1827](https://github.com/infrahq/infra/issues/1827)) ([c6d9932](https://github.com/infrahq/infra/commit/c6d9932682d3183edb2f2cb34d500ccaf47bed09))
* pin UI API version ([d1fdbc6](https://github.com/infrahq/infra/commit/d1fdbc603755f4b3e20d00a9b2a5355356c3b847))
* polymorphic id api migration ([f5286d6](https://github.com/infrahq/infra/commit/f5286d659873931c403c9134db8d34886ecd082c))
* remove `kubernetes.` prefix from destinations ([#1849](https://github.com/infrahq/infra/issues/1849)) ([bbecdf1](https://github.com/infrahq/infra/commit/bbecdf17ea76af59b0385c6bfb41b814179af1df))
* remove and update helm links ([#1866](https://github.com/infrahq/infra/issues/1866)) ([b8bc11f](https://github.com/infrahq/infra/commit/b8bc11f6d4c33c451a286653f5ecdcdb911e1e39))
* resolves issue with stacked response migrations ([#1928](https://github.com/infrahq/infra/issues/1928)) ([4e4c282](https://github.com/infrahq/infra/commit/4e4c2823ea62498800d1768a993527f2fe473527))
* revert "feat: add api versioning with request/response migrations ([#1884](https://github.com/infrahq/infra/issues/1884))" ([4cf2272](https://github.com/infrahq/infra/commit/4cf22726f8aee072939265bfee2ccc76a95254a9))
* revert ui's api version to 0.12.2 ([#1938](https://github.com/infrahq/infra/issues/1938)) ([02b94b1](https://github.com/infrahq/infra/commit/02b94b1ba1a202ade9e3890ada51d00cef6f98ca))
* safari ui support ([#1879](https://github.com/infrahq/infra/issues/1879)) ([0dcdd7e](https://github.com/infrahq/infra/commit/0dcdd7e5b215eeb9b215dca5e8ab0c98cc378382))
* serialize empty ID to 0 ([4c00719](https://github.com/infrahq/infra/commit/4c00719ec59749718def4cf501b405c0d74d8cd4))
* share modal input bugs ([#1842](https://github.com/infrahq/infra/issues/1842)) ([25501d8](https://github.com/infrahq/infra/commit/25501d8ac29279e125e902f74a0e4943702a5d2b))
* support redirecting previous path rewrites ([#1943](https://github.com/infrahq/infra/issues/1943)) ([569d1ca](https://github.com/infrahq/infra/commit/569d1ca344742cb34e745ac14a7f386d8539b1ab))
* **ui:** create admin grant ([5ee3fd0](https://github.com/infrahq/infra/commit/5ee3fd0d1066da9820e65136ed0c73e6e2a0103d))
* **ui:** UI does not show reset password page ([172b1d9](https://github.com/infrahq/infra/commit/172b1d971906b8b251b9bd6e4a110875cd868f40))
* **ui:** update to use grant.identity instead of grant.subject ([e8f075d](https://github.com/infrahq/infra/commit/e8f075d48a0d31711486262809c5256277d2c06f))
* undefined function ([84cfd53](https://github.com/infrahq/infra/commit/84cfd53f23c837fc283f6ffae50a6f9455ca2694))

### [0.12.2](https://github.com/infrahq/infra/compare/v0.12.1...v0.12.2) (2022-04-29)


### Features

* **cmd:** optional access key name ([56de3d1](https://github.com/infrahq/infra/commit/56de3d107c8250d6db9017fc2378ea24d9b41a7f))


### Bug Fixes

* block removing last infra admin ([ddf29a7](https://github.com/infrahq/infra/commit/ddf29a7647dcc225e18a936ab828f694ec003e26))
* do not label signup user grant as created by system ([5b900f4](https://github.com/infrahq/infra/commit/5b900f44752d28244c0d2ba5f24c4f678bc122d2))
* loading connector.Options ([a992e94](https://github.com/infrahq/infra/commit/a992e942495c56d052ee6e064c198df9983c2c3f))
* mac os uses symlink for canonical path ([#1792](https://github.com/infrahq/infra/issues/1792)) ([1c128c3](https://github.com/infrahq/infra/commit/1c128c31a6459865eb48aaf599e101ca2500cd13))

### [0.12.1](https://github.com/infrahq/infra/compare/v0.12.0...v0.12.1) (2022-04-28)


### Bug Fixes

* destinaion page ui bugs fixed ([#1747](https://github.com/infrahq/infra/issues/1747)) ([7239349](https://github.com/infrahq/infra/commit/723934993ae8eeb8e933c72340692d5262895f58))
* make generate ui ([#1759](https://github.com/infrahq/infra/issues/1759)) ([b7bd65a](https://github.com/infrahq/infra/commit/b7bd65afd054d32ecfaac3f6710aeed6cb1415a7))

## [0.12.0](https://github.com/infrahq/infra/compare/v0.11.1...v0.12.0) (2022-04-27)


### ⚠ BREAKING CHANGES

* removes the GET /v1/introspect endpoint. Use /v1/identities/self

### Bug Fixes

* revert pull request [#1682](https://github.com/infrahq/infra/issues/1682) from infrahq/dnephin/replace-viper ([c1cf195](https://github.com/infrahq/infra/commit/c1cf195c7607e4513231f4d856768b9dee8b2af8))
* update endpoints in ui ([61940a5](https://github.com/infrahq/infra/commit/61940a5a3231e4acb1816ee35c9d878837fed8f6))


### Improvement

* remove the /v1/introspect endpoint ([1c81e26](https://github.com/infrahq/infra/commit/1c81e26c1172128c2e141bfecb2cb51b0afb1a12))

### [0.11.1](https://github.com/infrahq/infra/compare/v0.11.0...v0.11.1) (2022-04-27)


### Features

* import identities through config ([8990200](https://github.com/infrahq/infra/commit/899020095ab0100cade1a6565149dea1d8703662))
* remove admin access key config ([d72ee95](https://github.com/infrahq/infra/commit/d72ee957eac6b0c0fc5fe2e12147518dedd07015))
* User interface refresh ([#1704](https://github.com/infrahq/infra/issues/1704)) ([4826baf](https://github.com/infrahq/infra/commit/4826bafedb1d8db140d72c500d80da454a503015))


### Bug Fixes

* add back local users to quickstart ([#1748](https://github.com/infrahq/infra/issues/1748)) ([5278f2f](https://github.com/infrahq/infra/commit/5278f2f01c4e1f5692df62779811b6667c5ef226))
* connector access key not found when connector is disabled ([51609b7](https://github.com/infrahq/infra/commit/51609b747bb7c6c440ad61cf31be08c7079330c6))
* do not look up k8s name when provided ([#1702](https://github.com/infrahq/infra/issues/1702)) ([95c8eff](https://github.com/infrahq/infra/commit/95c8effd7532e2f0d6bf3527d1bae2b09c989ccd))
* helm chart values.yaml for ui ([#1712](https://github.com/infrahq/infra/issues/1712)) ([1c63c03](https://github.com/infrahq/infra/commit/1c63c039713207297930443a322a85b513f2c0b4))
* only use color logging with a terminal ([d8d9a4d](https://github.com/infrahq/infra/commit/d8d9a4d58b4263befd6e9b53f7a32d156668cc71))
* use common ca instead of sni to generate connector certificates ([#1687](https://github.com/infrahq/infra/issues/1687)) ([43d4c3f](https://github.com/infrahq/infra/commit/43d4c3f710a8e6a3f2190d15bee2521d82da01bf))

## [0.11.0](https://github.com/infrahq/infra/compare/v0.10.3...v0.11.0) (2022-04-22)


### ⚠ BREAKING CHANGES

* refactor setup to signup
* the UI fields in the server config file have changed.

### Features

* add request timeouts ([#1594](https://github.com/infrahq/infra/issues/1594)) ([c744d1b](https://github.com/infrahq/infra/commit/c744d1b35764ecdb57f324b2f94e7480f5784fea))
* change signup request endpoint to use email ([#1662](https://github.com/infrahq/infra/issues/1662)) ([10c487e](https://github.com/infrahq/infra/commit/10c487ee1ac55319aa0ab16c075fafccdfa01d72))
* create admin user on first login ([a95e006](https://github.com/infrahq/infra/commit/a95e006b2cb7a8b6b3778b6a7788b03214293a86))
* format infra keys list with user name ([#1666](https://github.com/infrahq/infra/issues/1666)) ([5b87585](https://github.com/infrahq/infra/commit/5b8758582951f4f7b668d24291db14fa05d18065))
* login and logout of current or all servers, also affects clear ([#1633](https://github.com/infrahq/infra/issues/1633)) ([bd62622](https://github.com/infrahq/infra/commit/bd626220fc2c55fe51fc9c19e6ea22339282308a))


### Bug Fixes

* **api:** fix marshalling of api.Time values ([0299e4c](https://github.com/infrahq/infra/commit/0299e4cff9ff9026e0e930d6249cb1311b5f1564))
* **ci:** actions not running correctly ([ac5536b](https://github.com/infrahq/infra/commit/ac5536b5c28459433deaa6f73e5da160a269df89))
* cli log level and reading from env vars ([3947a69](https://github.com/infrahq/infra/commit/3947a69bc791100b986acf091d65993a47c6d0f5))
* **cli:** fix the extension-deadline flag ([d417c56](https://github.com/infrahq/infra/commit/d417c56a478c4e4dc87a05a0f3eea070b7550681))
* create provider user for signup user ([d53e508](https://github.com/infrahq/infra/commit/d53e5085ad8bd55b102c462d6ff578872fe6dbc4))
* do not set a otp on machine identities ([#1624](https://github.com/infrahq/infra/issues/1624)) ([e52b819](https://github.com/infrahq/infra/commit/e52b819be9333f5b601185459772cff5ebc7e146))
* fix 404 and proxy ui routes ([#1661](https://github.com/infrahq/infra/issues/1661)) ([5d59787](https://github.com/infrahq/infra/commit/5d59787dded278302f9e18dd879f4591646d9e8e))
* flakey test fix ([#1650](https://github.com/infrahq/infra/issues/1650)) ([5ce7dba](https://github.com/infrahq/infra/commit/5ce7dbafc1c6294ceacf5773dbde003c866df808))
* help prevent bad provider config ([#1630](https://github.com/infrahq/infra/issues/1630)) ([79bfca4](https://github.com/infrahq/infra/commit/79bfca4ae97f06b6a8899222f63e72293cb5d72e))


### Improvement

* refactor setup to signup ([adf34cf](https://github.com/infrahq/infra/commit/adf34cf04488671f0b2b51d6d0fadd3617851d44))
* structure the UI config ([b257bc9](https://github.com/infrahq/infra/commit/b257bc927b47bd53611fa34db1cd07a810d7880f))

### [0.10.3](https://github.com/infrahq/infra/compare/v0.10.2...v0.10.3) (2022-04-14)


### Features

* add about command ([#1573](https://github.com/infrahq/infra/issues/1573)) ([66063c4](https://github.com/infrahq/infra/commit/66063c4eb17a88bcad034bf64f9f230c30f1ee53))
* grants for inactive identities ([#1536](https://github.com/infrahq/infra/issues/1536)) ([#1564](https://github.com/infrahq/infra/issues/1564)) ([#1565](https://github.com/infrahq/infra/issues/1565)) ([5b98143](https://github.com/infrahq/infra/commit/5b98143614265f28faf5dc21974a1bec1795ea2d))


### Bug Fixes

* allow use by name alone ([5ef685a](https://github.com/infrahq/infra/commit/5ef685acc9ad1f0fba8e4cff74c4adc0cd50af21))
* docs build ([#1586](https://github.com/infrahq/infra/issues/1586)) ([3f7eebf](https://github.com/infrahq/infra/commit/3f7eebf4b5db32d00ce2b2a7b0bd8139453f24c9))
* fix examples of additionaSecrets ([d22c768](https://github.com/infrahq/infra/commit/d22c768cd8c2ec7582e32b3ac70a53a8304d274f))
* infra providers add set client ID/secret ([39228f8](https://github.com/infrahq/infra/commit/39228f8371a1c5de3ba7a8ee60553afec8e1a04f))
* loading of key providers from config ([8502dad](https://github.com/infrahq/infra/commit/8502dad963fd4348441df3aec62e38833ba37c9e))
* migrate identity provider_id with tests ([#1569](https://github.com/infrahq/infra/issues/1569)) ([709a37b](https://github.com/infrahq/infra/commit/709a37b3fc5bc81db706307cb7e5aaeccdf25675))
* Moved graphic to the top of page ([44ffe41](https://github.com/infrahq/infra/commit/44ffe416636fabdb77b110d42aee5e86b231b7fc))
* no provider lookup if zero ([ec5ffb2](https://github.com/infrahq/infra/commit/ec5ffb2b4d7c81ad1e3a6f5e82f3d959afde444d))
* not logging api calls ([ec220ff](https://github.com/infrahq/infra/commit/ec220ff680a80e379d7d590ef72315e073594392))
* re-index identities after de-duplication ([c46aa3e](https://github.com/infrahq/infra/commit/c46aa3e3aa15b33c3225c3b4ade8f313b448fc6f))
* **server:** use mapstructure to decode secrets config ([4053ccb](https://github.com/infrahq/infra/commit/4053ccbe8b4dbff1824c19bf12687153cfe15f88))
* version bump path ([d137a0d](https://github.com/infrahq/infra/commit/d137a0d2c9f8af6b62d94ad3d66511d4eca646e1))

### [0.10.2](https://github.com/infrahq/infra/compare/v0.10.1...v0.10.2) (2022-04-12)


### Bug Fixes

* db locking issue ([#1583](https://github.com/infrahq/infra/issues/1583)) ([fbafdbd](https://github.com/infrahq/infra/commit/fbafdbdb08be2dd37e574ee05ae8fb25493c237a))
* update docs for latest provider changes ([54a1af1](https://github.com/infrahq/infra/commit/54a1af1efe12ef242dd960af4777c6434ea39e23))

### [0.10.1](https://github.com/infrahq/infra/compare/v0.10.0...v0.10.1) (2022-04-12)


### Features

* set telemetry events ([#1421](https://github.com/infrahq/infra/issues/1421)) ([264cf85](https://github.com/infrahq/infra/commit/264cf85a9801989bb577f8178fe741be882e6844))


### Bug Fixes

* architecture image not rendering in docs on GitHub ([bc00aac](https://github.com/infrahq/infra/commit/bc00aac6f156dbf9218e6656252a008b980ec4dc))
* bring back roles in grant docs ([ccf34a8](https://github.com/infrahq/infra/commit/ccf34a8f6c019d3a5e24181c64be370902f9688b))
* broken doc links ([d9ea9a6](https://github.com/infrahq/infra/commit/d9ea9a68e198ec7e5dfa496f7bfa94d2c5abfcd9))
* ByIdentityKind uses non-existent column ([45ce7a7](https://github.com/infrahq/infra/commit/45ce7a7106e8db913ea112bafcef1b6b4715b9ca))
* cli example breaking docs ([06cae72](https://github.com/infrahq/infra/commit/06cae729dd1a48acaa6bde0e9b51749d2f991883))
* **cli:** edit providers add cmd from args to flags ([#1528](https://github.com/infrahq/infra/issues/1528)) ([12e7953](https://github.com/infrahq/infra/commit/12e79530d5a7bd23ac7738752e645e6eead58781))
* **cli:** updates 'grants add' command ([#1474](https://github.com/infrahq/infra/issues/1474)) ([bca65bf](https://github.com/infrahq/infra/commit/bca65bfb977485676836c3c6b9af3c312343e22a))
* do not display internal provider ([497d095](https://github.com/infrahq/infra/commit/497d095284bfc9224f45a7f7963689be9384779d))
* do not fail if group-user membership already exists ([#1567](https://github.com/infrahq/infra/issues/1567)) ([e50b45c](https://github.com/infrahq/infra/commit/e50b45ca39d969c83de11248362c88fd9926e02c))
* docs build ([4a18611](https://github.com/infrahq/infra/commit/4a186113efdf5da2e50f112a4ffe522424893488))
* **docs:** binary distribution links ([9787613](https://github.com/infrahq/infra/commit/9787613014c8ac7b203b9f301871e5ab091ae1e9))
* headings ([0d3101d](https://github.com/infrahq/infra/commit/0d3101dde98f3269ebcdeca6090e095a9fdca257))
* ids not showing up for infra ids list ([#1568](https://github.com/infrahq/infra/issues/1568)) ([4a339a9](https://github.com/infrahq/infra/commit/4a339a9e7053c8d8c3b392e35a2a325fc338bc43))
* infra keys command details ([#1501](https://github.com/infrahq/infra/issues/1501)) ([#1502](https://github.com/infrahq/infra/issues/1502)) ([#1506](https://github.com/infrahq/infra/issues/1506)) ([99fc0e8](https://github.com/infrahq/infra/commit/99fc0e807ee8d0bba06ac40d9b4586eb97c9491b))
* introduction.md ([#1578](https://github.com/infrahq/infra/issues/1578)) ([b4ee514](https://github.com/infrahq/infra/commit/b4ee51448709ce12cd03524c3940b6de285500b7))
* no error if failed to logout ([8e5dc57](https://github.com/infrahq/infra/commit/8e5dc5766c759fb4143439779f95b73e13a28907))
* prevent users from removing internal providers/identities ([df1e5ae](https://github.com/infrahq/infra/commit/df1e5ae51d46f02ffd319452d55d10789072b2e8))
* remove documentation for infra destinations add ([92b10cf](https://github.com/infrahq/infra/commit/92b10cf79ae887fa3a32414ec9101de4a76867a6))
* remove unneeded ol from docs ([2021904](https://github.com/infrahq/infra/commit/202190464b14b7d8296c4b5337d8c24578d79fbb))
* resolve issue with ambiguous optional selectors ([#1495](https://github.com/infrahq/infra/issues/1495)) ([b30c8ac](https://github.com/infrahq/infra/commit/b30c8ac20fb04c3d91f287ea82576b9416dcf567))
* set infra provider on created access key ([#1535](https://github.com/infrahq/infra/issues/1535)) ([a81fc05](https://github.com/infrahq/infra/commit/a81fc0532db33d890407412a5a0039cb41419f26))
* setup access key missing provider ID ([0a240ae](https://github.com/infrahq/infra/commit/0a240ae823c71ecae44cdf195abaea24717abdc2))
* show infra grants in list ([#1515](https://github.com/infrahq/infra/issues/1515)) ([#1533](https://github.com/infrahq/infra/issues/1533)) ([62147b2](https://github.com/infrahq/infra/commit/62147b263059c93893348cd5cd71a61f61134341))
* updates grant remove and list for launch ([#1547](https://github.com/infrahq/infra/issues/1547)) ([bfb0c64](https://github.com/infrahq/infra/commit/bfb0c6427473e37ff1b3cfc28a05d6c16b7801e6))

## [0.10.0](https://github.com/infrahq/infra/compare/v0.9.0...v0.10.0) (2022-04-07)


### ⚠ BREAKING CHANGES

* remove ununsed create token body (#1216) (#1497)
* remove destinations add/remove commands

### Features

* add back infra destinations remove ([19ee32f](https://github.com/infrahq/infra/commit/19ee32f3498e408b6153c52f8a24bad06da39070))
* always create admin/connector identities ([92fbff3](https://github.com/infrahq/infra/commit/92fbff33d10ccc350ebf29161d9689721b740faf))
* infra view role ([#1507](https://github.com/infrahq/infra/issues/1507)) ([00a47e1](https://github.com/infrahq/infra/commit/00a47e196f9279446ab0a0e8744bdab3eb887d5b))
* login connector to exchange access key ([bbd4f00](https://github.com/infrahq/infra/commit/bbd4f002d2f329e0382ec03f3a23bd6112796300))
* print no resource found for list commands ([97e73ee](https://github.com/infrahq/infra/commit/97e73ee9fc756d0b0eae05bfac49afa1daa49f70))


### Bug Fixes

* **api:** scope api middleware to api routes ([5ac180e](https://github.com/infrahq/infra/commit/5ac180e8f03cf8eff00cafa3ea558d55676e4969))
* do not clear config on empty config file ([#1456](https://github.com/infrahq/infra/issues/1456)) ([e38dcc1](https://github.com/infrahq/infra/commit/e38dcc1aa3fbcaeb01c4c902ef05c774c4afc329))
* remove destinations add/remove commands ([1f391b2](https://github.com/infrahq/infra/commit/1f391b24237a556dbe29733d5e7f1bc32df3a969))
* seperate docs action with correct commands ([2c8b501](https://github.com/infrahq/infra/commit/2c8b5013a8b33f07ddc53467ca941c1d4573f7b3))
* unit test ([9570413](https://github.com/infrahq/infra/commit/95704133b1fc560c924592f6416b71bf88cfd5fb))
* validate key names dont have spaces ([#1449](https://github.com/infrahq/infra/issues/1449)) ([#1490](https://github.com/infrahq/infra/issues/1490)) ([b080ca2](https://github.com/infrahq/infra/commit/b080ca206999d489dbcf0517cdb67552a44c5522))


### Maintenance

* remove ununsed create token body ([#1216](https://github.com/infrahq/infra/issues/1216)) ([#1497](https://github.com/infrahq/infra/issues/1497)) ([ac3c509](https://github.com/infrahq/infra/commit/ac3c50989d266faa5100b0d885fe7ff662c26fa0))

## [0.9.0](https://github.com/infrahq/infra/compare/v0.8.0...v0.9.0) (2022-04-06)


### ⚠ BREAKING CHANGES

* change LoginResponse polymorphicId to polymorphicID
* **api:** replaces /v1/user and /v1/machines with /v1/identities

### Features

* add expiry to login response ([484021b](https://github.com/infrahq/infra/commit/484021b26c231c1aac0923cadc6ffb7cb2453239))
* **cli:** check expiry for authenticated commands ([a9c23ef](https://github.com/infrahq/infra/commit/a9c23ef224c2577dd5be4c4ca9abb62672ad6d0e))


### Bug Fixes

* bump openapi to appropriate version ([c99fc3d](https://github.com/infrahq/infra/commit/c99fc3df8e309ab4aea131fda3ac6c5eeffc29ff))
* **cli:** check min requirement for new password ([#1435](https://github.com/infrahq/infra/issues/1435)) ([4777b9e](https://github.com/infrahq/infra/commit/4777b9ed2de14e948c8fcacc010a118dcc3fbc85))
* delete grants on user delete ([#1447](https://github.com/infrahq/infra/issues/1447)) ([26cb088](https://github.com/infrahq/infra/commit/26cb08898c69a3c4a5ef001d17cec9b6c63fa167))
* do not allow users to delete themselves ([#1473](https://github.com/infrahq/infra/issues/1473)) ([4de92e4](https://github.com/infrahq/infra/commit/4de92e45fec7186ee8ba55820cb1324dbb955014))


### improve

* **api:** unify users and machine ([c76073d](https://github.com/infrahq/infra/commit/c76073dc6fced90595ffc86aaf85b3582eac657a))


### maintain

* use json tag for property names ([5fa6413](https://github.com/infrahq/infra/commit/5fa64136da43045b84a8593d24b927cdf588f1fd))

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
