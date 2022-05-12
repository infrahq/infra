const fetch = global.fetch

const base = '0.12.0'

// Patch the global fetch to include our base API
// version for requests to the same domain
global.fetch = (resource, info) => fetch(resource, {
  ...info,
  ...resource.startsWith('/')
    ? {
        headers: {
          'Infra-Version': base
        }
      }
    : {}
})
