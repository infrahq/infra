import Cookies from 'universal-cookie'

export function saveToVisitedOrgs(domain, orgName) {
  const cookies = new Cookies()

  let visitedOrgs = cookies.get('orgs') || []

  if (!visitedOrgs.find(x => x.url === domain)) {
    visitedOrgs.push({
      url: domain,
      name: orgName,
    })

    cookies.set('orgs', visitedOrgs, {
      path: '/',
      domain: `.${currentBaseDomain()}`,
    })
  }
}

export function currentBaseDomain() {
  let parts = window.location.host.split('.')
  if (parts.length > 2) {
    parts.shift() // remove the org
  }
  return parts.join('.') // return the domain without the org
}

export function currentOrg() {
  let parts = window.location.host.split('.')
  if (parts.length > 2) {
    return parts.shift() // this is the org
  }
  return ''
}

export function persistLoginRedirectCookie(orgName) {
  const cookies = new Cookies()

  // set the cookie domain to a general base domain
  let cookieDomain = currentBaseDomain()

  cookies.set('finishLogin', orgName, {
    path: '/',
    domain: `.${cookieDomain}`,
    sameSite: 'lax',
  })
}
