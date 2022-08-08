export const descriptions = {
  'cluster-admin': 'Super-user access to perform any action on any resource',
  admin: 'Read and write access to all resources',
  edit: 'Read and write access to most resources, but not roles',
  view: 'Read-only access to see most resources',
  logs: 'Read and stream logs',
  exec: 'Shell to a running container',
  'port-forward': 'Use port-forwarding to access applications',
}

export function sortByPrivilege(a, b) {
  if (a?.privilege === 'cluster-admin') {
    return -1
  }

  if (b?.privilege === 'cluster-admin') {
    return 1
  }

  return a?.privilege?.localeCompare(b?.privilege)
}

export function sortByHasDescriptions(a, b) {
  const descriptionsList = Object.keys(descriptions)

  if (descriptionsList.includes(a)) {
    return -1
  }

  if (descriptionsList.includes(b)) {
    return 1
  }

  return a.localeCompare(b)
}

export function sortByResource(a, b) {
  return a?.resource?.localeCompare(b?.resource)
}

export function sortBySubject(a, b) {
  return (a?.user || a?.group)?.localeCompare(b?.user || b?.group)
}
