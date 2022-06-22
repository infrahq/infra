import useSWR, { useSWRConfig } from 'swr'

// multiFetcher is an SWR fetcher that can fetch multiple urls instead of just one
// this is used for fetching grants for multiple groups simultaneously
function multiFetcher (...urls) {
  return Promise.all(urls.map(url => fetch(url).then(r => r.json())))
}

function buildquery ({ resource, user, group, privilege }) {
  const query = new URLSearchParams()

  if (resource) {
    query.append('resource', resource)
  }

  if (user) {
    query.append('user', user)
  }

  if (group) {
    query.append('group', group)
  }

  if (privilege) {
    query.append('privilege', privilege)
  }

  return query
}

// useGrants is a custom hook that returns a "complete" list of grants including grants inherited via:
// 1. access to parent resources
// 2. access due to group membership
export function useGrants ({ resource, user, group, privilege, hideInfra = false }) {
  // fetch grants with the exact query
  const query = `/api/grants?${buildquery({ resource, user, group, privilege })}`
  const { data: { items } = {} } = useSWR(query)

  // fetch inherited grants from parent resources
  const parts = resource?.split('.') || []
  const shouldFetchParent = parts.length > 1
  const { data: { items: parentGrants } = {} } = useSWR(shouldFetchParent ? `/api/grants?${buildquery({ resource: parts[0], user, group, privilege })}` : null)
  const inheritedByParents = parentGrants?.map(g => ({ ...g, inherited: true }))

  // pre-fill user and group information
  // todo: allow for expansions in the API to avoid this client-side
  const { data: { items: users } = {} } = useSWR(resource ? '/api/users' : null)
  const { data: { items: groups } = {} } = useSWR(resource ? '/api/groups' : null)

  // fetch inherited grants from group membership
  // todo: move this to api since it requires fetching the grants for each of the user's groups
  const { data: { items: userGroups } = {} } = useSWR(user ? `/api/groups?userID=${user}` : null)
  const { data: groupGrants } = useSWR(() => userGroups ? userGroups.map(ug => `/api/grants?${buildquery({ resource, group: ug.id, privilege })}`) : null, multiFetcher)
  const inheritedByGroups = groupGrants?.map(gg => gg.items)?.flat()?.map(g => ({ ...g, inherited: true })) || []

  const { mutate } = useSWRConfig()

  if (!items) {
    return {}
  }

  if (shouldFetchParent && !inheritedByParents) {
    return {}
  }

  if (user && (!userGroups || !inheritedByGroups)) {
    return {}
  }

  // Merge all grants into a single list and sort them by:
  // 1. non-inherited first
  // 2. group-first
  // 3. user or group ID (alphabetically)
  const all = [...items || [], ...inheritedByParents || [], ...inheritedByGroups || []].sort((a, b) => {
    if (a.inherited && !b.inherited) {
      return 1
    }

    if (b.inherited && !a.inherited) {
      return -1
    }

    // then prioritize groups
    if (a.group && !b.group) {
      return -1
    }

    if (b.group && !a.group) {
      return 1
    }

    // then sort by user or group ID for
    // consistent positions between updates
    if (resource) {
      const left = a.user || a.group || ''
      const right = b.user || b.group || ''
      return left.localeCompare(right)
    }

    return a?.resource?.localeCompare(b?.resource)
  })

  // Add additional fields to the default group objects:
  // user: the user object
  // group: the group object
  // edit: a function to change the grant
  // delete: a function to delete the grant
  const grants = all.map(grant => {
    function edit (role) {
      // don't edit inherited grants
      if (grant.inherited || role === grant.privilege) {
        return
      }

      mutate(
        query,
        async ({ items: grants } = { items: [] }) => {
          // create new grant
          const res = await fetch('/api/grants', {
            method: 'POST',
            body: JSON.stringify({ ...grant, privilege: role })
          })

          const data = await res.json()

          if (!res.ok) {
            throw data
          }

          // delete old grant
          await fetch(`/api/grants/${grant.id}`, { method: 'DELETE' })

          return { items: [...grants.filter(g => g.id !== grant.id), data] }
        })
    }

    function remove () {
      // don't delete inherited grants
      if (grant.inherited) {
        return
      }

      mutate(
        query,
        async ({ items: grants } = { items: [] }) => {
          await fetch(`/api/grants/${grant.id}`, { method: 'DELETE' })
          return { items: grants?.filter(item => item?.id !== grant.id) }
        }, {
          optimisticData: { items: items?.filter(item => item.id !== grant.id) }
        })
    }

    const user = users?.find(u => u.id === grant.user) || {}
    const group = groups?.find(g => g.id === grant.group) || {}

    return {
      ...grant,
      user,
      group,
      edit,
      remove
    }
  }).filter(g => g?.user?.name !== 'connector')

  if (hideInfra) {
    return {
      grants: grants?.filter(grant => grant.resource !== 'infra') || [],
      loading: false
    }
  }

  return {
    grants,
    loading: false
  }
}
