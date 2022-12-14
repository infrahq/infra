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

export function formatPasswordRequirements(requirements) {
  return (
    'needs at least ' +
    requirements.reduce((value, currentValue, currentIndex) => {
      return (
        value +
        (currentIndex === requirements.length - 1
          ? requirements.length > 2
            ? ', and '
            : ' and '
          : ', ') +
        currentValue
      )
    })
  )
}
