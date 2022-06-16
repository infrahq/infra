module.exports = {
  async redirects () {
    return [
      {
        source: '/docs/:slug*.md',
        destination: '/docs/:slug*',
        permanent: true
      },
      {
        source: '/docs/guides/identity-providers/:slug*',
        destination: '/docs/identity-providers/:slug*',
        permanent: true
      },
      {
        source: '/docs/install/configure/custom-domain',
        destination: '/docs/install/custom-domain',
        permanent: true
      },
      {
        source: '/docs/install/configure/encryption',
        destination: '/docs/reference/helm-reference#encryption',
        permanent: true
      },
      {
        source: '/docs/install/configure/postgres',
        destination: '/docs/reference/helm-reference#postgres-database',
        permanent: true
      },
      {
        source: '/docs/install/configure/secrets',
        destination: '/docs/reference/helm-reference#secrets',
        permanent: true
      },
      {
        source: '/docs/install/configure/custom-domain',
        destination: '/docs/install/custom-domain',
        permanent: true
      },
      {
        source: '/docs/install/configure/custom-domain',
        destination: '/docs/install/custom-domain',
        permanent: true
      },
      {
        source: '/docs/guides/:slug*',
        destination: '/docs/configuration/:slug*',
        permanent: true
      },
      {
        source: '/docs',
        destination: '/docs/getting-started',
        permanent: true
      },
      {
        source: '/docs/getting-started/key-concepts',
        destination: '/docs/reference/how-infra-works',
        permanent: true
      }
    ]
  },
  async headers () {
    return [
      {
        source: '/_next/image(.*)',
        headers: [
          {
            key: 'Cache-Control',
            value: 'public, max-age=315360000, immutable'
          }
        ]
      }
    ]
  }
}
