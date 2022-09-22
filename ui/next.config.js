const { PHASE_DEVELOPMENT_SERVER } = require('next/constants')

module.exports = phase => ({
  reactStrictMode: true,
  generateBuildId: async () => {
    if (process.env.NEXT_BUILD_ID) {
      return process.env.NEXT_BUILD_ID
    }

    return null
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
              phase === PHASE_DEVELOPMENT_SERVER ? '' : `default-src 'self'`,
          },
        ],
      },
    ]
  },
  output: 'standalone',
})
