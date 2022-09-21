export const descriptions = {
  'cluster-admin': 'Super-user access to perform any action on any resource',
  admin: 'Read and write access to all resources',
  edit: 'Read and write access to most resources, but not roles',
  view: 'Read-only access to see most resources',
  exec: 'Shell to a running container',
  'port-forward': 'Use port-forwarding to access applications',
  logs: 'Read and stream logs',
}

export function sortByPrivilege(a, b) {
  if (a?.privilege === 'cluster-admin') {
    return -1
  }

  if (b?.privilege === 'cluster-admin') {
    return 1
  }

  return 0
}

export function sortedByPrivilegeArray(list) {
  const sortedArray = Object.keys(descriptions)?.filter(
    Set.prototype.has,
    new Set(list)
  )

  const difference = list.filter(x => !sortedArray.includes(x))

  const arrayMap = list.reduce(
    (accumulator, currentValue) => ({
      ...accumulator,
      [currentValue]: currentValue,
    }),
    {}
  )

  return [...sortedArray.map(key => arrayMap[key]), ...difference]
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
