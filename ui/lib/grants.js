export async function editGrant (id, { user, group, privilege, resource }) {
  const res = await fetch('/api/grants', {
    method: 'POST',
    body: JSON.stringify({
      user,
      group,
      privilege,
      resource
    })
  })

  const data = await res.json()
  if (!res.ok) {
    throw data
  }

  // delete old grant
  await fetch(`/api/grants/${id}`, { method: 'DELETE' })

  return data
}

export function sortByPrivilege (a, b) {
  if (a === 'cluster-admin') {
    return -1
  }

  if (b === 'cluster-admin') {
    return 1
  }

  return a?.privilege?.localeCompare(b?.privilege)
}

export function sortByResource (a, b) {
  return a?.resource?.localeCompare(b?.resource)
}

export function sortBySubject (a, b) {
  return (a?.user || a?.group)?.localeCompare(b?.user || b?.group)
}
