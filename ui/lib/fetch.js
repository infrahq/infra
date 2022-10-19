const fetch = global.fetch

const base = '0.13.0'

// Patch the global fetch to include our base API
// version for requests to the same domain
global.fetch = (resource, info) =>
  fetch(resource, {
    ...(resource.startsWith('/')
      ? {
          headers: {
            'Infra-Version': base,
          },
        }
      : {}),
    ...info,
  })

// jsonBody returns a js object or throws an error matching the {code: x, message: y} format, where x is a number and y is a string.
global.jsonBody = async res => {
  if (!res.ok) {
    // check if response is json before trying to parse it
    const text = await res.text()
    if (text.length > 1 && text[0] == '{') throw await JSON.parse(text)
    else throw { code: res.status, message: res.statusText }
  }

  return res.json()
}
