export const descriptions = {
  'cluster-admin': 'Super-user access to perform any action on any resource',
  admin: 'Read and write access to all resources',
  edit: 'Read and write access to most resources, but not roles',
  view: 'Read-only access to see most resources',
  exec: 'Shell to a running container',
  'port-forward': 'Use port-forwarding to access applications',
  logs: 'Read and stream logs',
}

const KUBERNETES_ROLE_ORDER = [
  'cluster-admin',
  'admin',
  'edit',
  'view',
  'exec',
  'port-forward',
  'logs',
]

export function sortByPrivilege(a, b) {
  if (a?.privilege === 'cluster-admin') {
    return -1
  }

  if (b?.privilege === 'cluster-admin') {
    return 1
  }

  return 0
}

export function sortByRole(list = []) {
  const sortedList = list
    .filter(x => KUBERNETES_ROLE_ORDER.includes(x))
    .sort(
      (a, b) =>
        KUBERNETES_ROLE_ORDER.indexOf(a) - KUBERNETES_ROLE_ORDER.indexOf(b)
    )

  const difference = list
    .filter(x => !sortedList.includes(x))
    .sort((a, b) => a.localeCompare(b))

  return [...sortedList, ...difference]
}

export function sortBySubject(a, b) {
  if (a?.user && b?.group) {
    return -1
  }

  if (a?.group && b?.user) {
    return 1
  }

  return 0
}
