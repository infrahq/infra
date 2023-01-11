const { PHASE_DEVELOPMENT_SERVER } = require('next/constants')

module.exports = phase => ({
  reactStrictMode: true,
  generateBuildId: async () => {
    if (process.env.NEXT_BUILD_ID) {
      return process.env.NEXT_BUILD_ID
    }

    return null
  },
  async redirects() {
    return [
      {
        source: '/',
        has: [
          {
            type: 'cookie',
            key: 'auth',
          },
        ],
        destination: '/destinations',
        permanent: false,
      },
    ]
  },
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
                : `default-src 'self'; img-src * 'self' data: https:;`,
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
  output: 'standalone',
})
