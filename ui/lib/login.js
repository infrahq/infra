export function saveToVisitedOrgs(domain, orgName) {
  const cookies = new Cookies()

  let visitedOrgs = cookies.get('orgs') || []

  if (!visitedOrgs.find(x => x.url === domain)) {
    visitedOrgs.push({
      url: domain,
      name: orgName,
    })

    // set the cookie domain to a general base domain
    let cookieDomain = window.location.host
    let parts = cookieDomain.split('.')
    if (parts.length > 2) {
      parts.shift() // remove the org
      cookieDomain = parts.join('.') // join the last two parts of the domain
    }

    cookies.set('orgs', visitedOrgs, {
      path: '/',
      domain: `.${cookieDomain}`,
    })
  }
}
