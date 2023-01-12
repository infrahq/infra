const { PHASE_DEVELOPMENT_SERVER } = require('next/constants')

const ContentSecurityPolicy = `
  base-uri 'none';
  default-src 'none';
  connect-src 'self';
  font-src 'self';
  img-src 'self' data:;
  prefetch-src 'self';
  script-src 'self';
  style-src 'self';
  frame-ancestors 'none';
  form-action 'self';
`

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
  output: 'standalone',
})
