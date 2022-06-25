export function addGrant ({ user, group, privilege, resource }) {
  return async ({ items: grants } = { items: [] }) => {
    // grant already exists, don't create it
    for (const g of grants) {
      if (g.privilege === privilege && g.user === user && g.group === group) {
        return { items: grants }
      }
    }

    // create new grant
    const res = await fetch('/api/grants', {
      method: 'POST',
      body: JSON.stringify({ user, group, privilege, resource })
    })

    const data = await res.json()

    if (!res.ok) {
      throw data
    }

    return { items: [...grants, data] }
  }
}

export function editGrant (id, { user, group, privilege, resource }) {
  return async ({ items: grants } = { items: [] }) => {
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

    return { items: [...grants.filter(g => id !== g.id), data] }
  }
}

export function removeGrant (id) {
  return async ({ items: grants } = { items: [] }) => {
    await fetch(`/api/grants/${id}`, { method: 'DELETE' })
    return { items: grants?.filter(g => g?.id !== id) }
  }
}

export function sortBySubject (a, b) {
  return (a?.user || a?.group)?.localeCompare(b?.user || b?.group)
}

export function sortByPrivilege (a, b) {
  return a?.privilege?.localeCompare(b?.privilege)
}

export function sortByResource (a, b) {
  return a?.resource?.localeCompare(b?.resource)
}
