const { PHASE_DEVELOPMENT_SERVER } = require('next/constants')

const ContentSecurityPolicy = `
  default-src 'self';
  script-src 'self' www.google.com www.gstatic.com;
  style-src 'self' 'unsafe-inline';
  img-src 'self' user-images.githubusercontent.com raw.githubusercontent.com i.ytimg.com;
  frame-src www.google.com youtube.com www.youtube.com youtube-nocookie.com www.youtube-nocookie.com;
  connect-src 'self' api.segment.io cdn.segment.com;
`

module.exports = phase => ({
  async headers() {
    return [
      {
        // Apply these headers to all routes in your application.
        source: '/:path*',
        headers: [
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          {
            key: 'Content-Security-Policy',
            value:
              phase === PHASE_DEVELOPMENT_SERVER
                ? ''
                : ContentSecurityPolicy.replace(/\s{2,}/g, ' ').trim(),
          },
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
          {
            key: 'Referrer-Policy',
            value: 'same-origin',
          },
          {
            key: 'X-XSS-Protection',
            value: '1; mode=block',
          },
        ],
      },
    ]
  },
  async redirects() {
    return [
      {
        source: '/docs/:slug*.md',
        destination: '/docs/:slug*',
        permanent: true,
      },

      {
        source: '/docs',
        destination: '/docs/start/what-is-infra',
        permanent: true,
      },
      ...[
        '/docs/guides/identity-providers/:slug*',
        '/docs/configuration/identity-providers/:slug*',
        '/docs/identity-providers/:slug*',
      ].map(source => {
        return {
          source,
          destination: '/docs/manage/idp/:slug*',
          permanent: true,
        }
      }),

      ...[
        '/docs/getting-started/introduction',
        '/docs/getting-started/what-is-infra',
        '/docs/getting-started/key-concepts',
        '/docs/reference/how-infra-works',
      ].map(source => {
        return {
          source,
          destination: '/docs/start/what-is-infra',
          permanent: true,
        }
      }),
      ...[
        '/docs/getting-started/quickstart',
        '/docs/getting-started/deploy',
        '/docs/install/install-on-kubernetes',
      ].map(source => {
        return {
          source,
          destination: '/docs/start/quickstart',
          permanent: true,
        }
      }),
      ...[
        '/docs/install/configure/encryption',
        '/docs/reference/helm-reference#encryption',
      ].map(source => {
        return {
          source,
          destination: '/docs/reference/helm#encryption',
          permanent: true,
        }
      }),
      ...[
        '/docs/install/configure/postgres',
        '/docs/reference/helm-reference#postgres-database',
      ].map(source => {
        return {
          source,
          destination: '/docs/reference/helm#postgres-database',
          permanent: true,
        }
      }),
      ...[
        '/docs/reference/cli-reference',
      ].map(source => {
        return {
          source,
          destination: '/docs/reference/cli',
          permanent: true,
        }
      }),
      ...[
        '/docs/getting-started/key-concepts',
        '/docs/reference/how-infra-works',
      ].map(source => {
        return {
          source,
          destination: '/docs/reference/architecture',
          permanent: true,
        }
      }),
      ...[
        '/docs/install/install-infra-cli',
        '/docs/getting-started/install-infra-cli',
      ].map(source => {
        return {
          source,
          destination: '/docs/start/install-infra-cli',
          permanent: true,
        }
      }),
      ...[
        '/docs/install/configure/secrets',
        '/docs/reference/helm-reference#secrets',
      ].map(source => {
        return {
          source,
          destination: '/docs/reference/helm#secrets',
          permanent: true,
        }
      }),
      ...['/docs/guides/:slug*', '/docs/configuration/:slug*'].map(source => {
        return {
          source,
          destination: '/docs/using/:slug*',
          permanent: true,
        }
      }),
      ...['/docs/install/upgrade', '/docs/getting-started/upgrade'].map(
        source => {
          return {
            source,
            destination: '/docs/start/upgrade',
            permanent: true,
          }
        }
      ),
      {
        source: '/docs/install/configure/custom-domain',
        destination: '/docs',
        permanent: true,
      },
      {
        source: '/terms',
        destination:
          'https://infrahq.notion.site/Terms-of-Service-6f3a635c638f4cb59f04df509208b1a3',
        permanent: false,
      },
      {
        source: '/privacy',
        destination:
          'https://infrahq.notion.site/Privacy-Policy-1b320c4f95904f9a83931d01a326a10b',
        permanent: false,
      },
    ]
  },
  images: {
    domains: ['raw.githubusercontent.com', 'user-images.githubusercontent.com'],
  },
  experimental: { images: { allowFutureImage: true } },
})
