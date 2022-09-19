module.exports = {
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
        return { source, destination: '/docs/idp/:slug*', permanent: true }
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
      {
        source: '/docs/getting-started/quickstart',
        destination: '/docs/getting-started/deploy',
        permanent: true,
      },
      ...[
        '/docs/getting-started/quickstart',
        '/docs/getting-started/deploy',
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
      {
        source: '/docs/getting-started/key-concepts',
        destination: '/docs/reference/how-infra-works',
        permanent: true,
      },

      {
        source: '/docs/install/install-on-kubernetes',
        destination: '/docs/getting-started/deploy',
        permanent: true,
      },
      {
        source: '/docs/install/install-infra-cli',
        destination: '/docs/getting-started/install-infra-cli',
        permanent: true,
      },
      {
        source: '/docs/install/upgrade',
        destination: '/docs/getting-started/upgrade',
        permanent: true,
      },
    ]
  },
  images: {
    domains: ['raw.githubusercontent.com', 'user-images.githubusercontent.com'],
  },
}
