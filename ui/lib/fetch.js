const fetch = global.fetch

// Patch the global fetch to include our base API
// version for requests to the same domain
global.fetch = version('0.12.0')

function version (version) {
  return (resource, info) => fetch(resource, {
    ...info,
    ...resource.startsWith('/')
      ? {
          headers: {
            'Infra-Version': version
          }
        }
      : {}
  })
}
